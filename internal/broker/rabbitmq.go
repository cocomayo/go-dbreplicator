package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go-dbreplicator/internal/engine"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	conn  *amqp.Connection
	ch    *amqp.Channel
	queue amqp.Queue
}

// NewRabbitMQ establishes the connection and ensures the queue exists.
func NewRabbitMQ(host string, port int, queueName string) (*RabbitMQ, error) {
	connStr := fmt.Sprintf("amqp://guest:guest@%s:%d/", host, port)
	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Durable queue so messages survive a RabbitMQ server restart
	q, err := ch.QueueDeclare(queueName, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	return &RabbitMQ{conn: conn, ch: ch, queue: q}, nil
}

// SaveData is used by the Producer to publish extracted records to the queue.
func (r *RabbitMQ) SaveData(ctx context.Context, records []engine.Record) error {
	for _, record := range records {
		body, err := json.Marshal(record)
		if err != nil {
			return err
		}

		err = r.ch.PublishWithContext(ctx, "", r.queue.Name, false, false, amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// FetchData is used by the Consumer to pull records from the queue.
func (r *RabbitMQ) FetchData(ctx context.Context, since time.Time) ([]engine.Record, error) {
	var records []engine.Record
	batchSize := 100 // Maximum messages to pull per tick to avoid overloading memory

	for i := 0; i < batchSize; i++ {
		msg, ok, err := r.ch.Get(r.queue.Name, false)
		if err != nil {
			return nil, err
		}
		if !ok {
			break // The queue is empty
		}

		var record engine.Record
		if err := json.Unmarshal(msg.Body, &record); err != nil {
			msg.Nack(false, false) // Reject unreadable messages
			continue
		}

		records = append(records, record)
		msg.Ack(false) // Acknowledge successful processing so RabbitMQ deletes it
	}

	return records, nil
}

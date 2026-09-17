package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-dbreplicator/internal/broker"
	"go-dbreplicator/internal/config"
	"go-dbreplicator/internal/engine"
	"go-dbreplicator/internal/factory"
)

func main() {
	configPath := flag.String("config", "data/config.yaml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 1. Source: Real DB or Mock File
	src, err := factory.NewSource(cfg.Source)
	if err != nil {
		log.Fatalf("Failed to initialize source: %v", err)
	}

	// 2. Destination: RabbitMQ
	dest, err := broker.NewRabbitMQ(cfg.Broker.Host, cfg.Broker.Port, cfg.Broker.TopicOrQueue)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// 3. Engine uses a state file to track the database timestamps
	producerEngine := engine.NewEngine(src, dest, cfg.PollingInterval, "data/producer_state.json")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("Starting Producer...")
	producerEngine.Run(ctx)
}

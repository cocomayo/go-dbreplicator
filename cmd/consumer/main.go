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

	// 1. Source: RabbitMQ
	src, err := broker.NewRabbitMQ(cfg.Broker.Host, cfg.Broker.Port, cfg.Broker.TopicOrQueue)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// 2. Destination: Real DB or Mock File
	dest, err := factory.NewDestination(cfg.Destination)
	if err != nil {
		log.Fatalf("Failed to initialize destination: %v", err)
	}

	// 3. Engine uses a dummy state file (Consumer doesn't query by timestamp, but the engine requires the parameter)
	consumerEngine := engine.NewEngine(src, dest, cfg.PollingInterval, "data/consumer_state.json")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Println("Starting Consumer...")
	consumerEngine.Run(ctx)
}

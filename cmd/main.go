package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-dbreplicator/internal/config"
	"go-dbreplicator/internal/engine"
	"go-dbreplicator/internal/factory"
)

func main() {
	// Define command-line flags with default fallback values
	configPath := flag.String("config", "data/config.yaml", "Path to the configuration JSON file")
	statePath := flag.String("state", "data/state.json", "Path to the state JSON file")

	// Parse the flags before using them
	flag.Parse()

	log.Println("Initializing DB Replicator...")

	// 1. Load the Application Configuration using the flag pointer
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// 2. Initialize Dependencies
	src, err := factory.NewSource(cfg.Source)
	if err != nil {
		log.Fatalf("Failed to initialize source: %v", err)
	}

	dest, err := factory.NewDestination(cfg.Destination)
	if err != nil {
		log.Fatalf("Failed to initialize destination: %v", err)
	}

	// 3. Create Engine using the dynamic state path
	replicationEngine := engine.NewEngine(
		src,
		dest,
		cfg.PollingInterval,
		*statePath,
	)

	// 4. Setup Shutdown Context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 5. Run
	replicationEngine.Run(ctx)
	log.Println("Replicator stopped cleanly.")
}

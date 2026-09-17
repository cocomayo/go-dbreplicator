package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"go-dbreplicator/internal/broker"
	"go-dbreplicator/internal/config"
	"go-dbreplicator/internal/engine"
	"go-dbreplicator/internal/factory"
)

func main() {
	configPath := flag.String("config", "data/config.yaml", "Path to infrastructure config")
	// Update flag to accept multiple, comma-separated use cases
	useCasesFlag := flag.String("use-cases", "", "Comma-separated business use cases (e.g., 'pintar/customers,hr/employees')")
	flag.Parse()

	if *useCasesFlag == "" {
		log.Fatal("At least one --use-cases must be provided")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	useCases := strings.Split(*useCasesFlag, ",")
	var wg sync.WaitGroup

	log.Printf("Starting Producer for %d use cases...", len(useCases))

	for _, uc := range useCases {
		useCase := strings.TrimSpace(uc)
		if useCase == "" {
			continue
		}

		wg.Add(1)

		// Launch each engine in its own Goroutine
		go func(uc string) {
			defer wg.Done()

			// Split "pintar/kaskel_transaction" into tenant ("pintar") and name ("kaskel_transaction")
			parts := strings.SplitN(uc, "/", 2)
			if len(parts) != 2 {
				log.Printf("[Skip %s] Invalid use case format. Expected 'tenant/name' (e.g., 'pintar/kaskel_transaction')", uc)
				return
			}
			tenant, name := parts[0], parts[1]

			// 1. Resolve SQL Query path based on your new directory structure
			queryPath := fmt.Sprintf("data/%s/queries/%s.sql", tenant, name)
			queryBytes, err := os.ReadFile(queryPath)
			if err != nil {
				log.Printf("[Skip %s] Failed to read query at %s: %v", uc, queryPath, err)
				return
			}

			// 2. Initialize Source
			src, err := factory.NewSource(cfg.Source, string(queryBytes))
			if err != nil {
				log.Printf("[Skip %s] Failed to initialize source: %v", uc, err)
				return
			}

			// 3. Initialize Destination (RabbitMQ)
			safeName := strings.ReplaceAll(uc, "/", "_")
			queueName := fmt.Sprintf("%s_%s", cfg.Broker.TopicOrQueue, safeName)

			dest, err := broker.NewRabbitMQ(cfg.Broker.Host, cfg.Broker.Port, queueName)
			if err != nil {
				log.Printf("[Skip %s] Failed to connect to RabbitMQ: %v", uc, err)
				return
			}

			// 4. Isolated State File
			stateFile := fmt.Sprintf("data/state_producer_%s.json", safeName)

			producerEngine := engine.NewEngine(src, dest, cfg.PollingInterval, stateFile)

			log.Printf("[%s] Engine running. Queue: %s", uc, queueName)
			producerEngine.Run(ctx)
		}(useCase)
	}

	// Block main thread until OS interruption (Ctrl+C) cancels the context and engines shut down
	wg.Wait()
	log.Println("All producer engines shut down successfully.")
}

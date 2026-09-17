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
	"go-dbreplicator/internal/mapper"
)

func main() {
	configPath := flag.String("config", "data/config.yaml", "Path to infrastructure config file")
	useCasesFlag := flag.String("use-cases", "", "Comma-separated business use cases (e.g., 'pintar/kaskel_transaction')")
	flag.Parse()

	if *useCasesFlag == "" {
		log.Fatal("At least one --use-cases flag must be provided")
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	useCases := strings.Split(*useCasesFlag, ",")
	var wg sync.WaitGroup

	log.Printf("Starting Consumer for %d use cases...", len(useCases))

	for _, uc := range useCases {
		useCase := strings.TrimSpace(uc)
		if useCase == "" {
			continue
		}

		wg.Add(1)

		go func(uc string) {
			defer wg.Done()

			// Split use case into tenant and resource name (e.g., "pintar/kaskel_transaction")
			parts := strings.SplitN(uc, "/", 2)
			if len(parts) != 2 {
				log.Printf("[Skip %s] Invalid format. Expected 'tenant/name' (e.g., 'pintar/kaskel_transaction')", uc)
				return
			}
			tenant, name := parts[0], parts[1]

			safeName := strings.ReplaceAll(uc, "/", "_")
			queueName := fmt.Sprintf("%s_consumer_%s", cfg.Broker.TopicOrQueue, safeName)

			// 1. Initialize Source (RabbitMQ Queue specific to this use case)
			src, err := broker.NewRabbitMQ(cfg.Broker.Host, cfg.Broker.Port, queueName)
			if err != nil {
				log.Printf("[Skip %s] Failed to connect to RabbitMQ: %v", uc, err)
				return
			}

			// 2. Initialize Destination (Base Destination Database)
			dest, err := factory.NewDestination(cfg.Destination)
			if err != nil {
				log.Printf("[Skip %s] Failed to initialize destination: %v", uc, err)
				return
			}

			// 3. Load Mapping Rules from data/{tenant}/mapper/from-{tenant}-{name}.yaml
			mapperPath := fmt.Sprintf("data/monitoring-tool/mapper/%v/%s.yaml", tenant, name)
			mappingRules, err := mapper.Load(mapperPath)
			if err != nil {
				log.Printf("[Skip %s] Missing or invalid mapping at %s: %v", uc, mapperPath, err)
				return
			}

			// Wrap destination with the MappedDestination decorator to handle 1:1 and JSON free_text mapping
			dest = &mapper.MappedDestination{
				Base:    dest,
				Mapping: mappingRules,
			}
			log.Printf("[%s] Loaded mapping rules from %s", uc, mapperPath)

			// 4. Isolated State File for High-Water Mark tracking
			stateFile := fmt.Sprintf("data/state_consumer_%s.json", safeName)

			consumerEngine := engine.NewEngine(src, dest, cfg.PollingInterval, stateFile)

			log.Printf("[%s] Consumer engine running. Listening on queue: %s", uc, queueName)
			consumerEngine.Run(ctx)
		}(useCase)
	}

	// Block main thread until an interrupt signal is received and all goroutines finish
	wg.Wait()
	log.Println("All consumer engines shut down successfully.")
}

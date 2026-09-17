package engine

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// Record is the universal intermediate format.
type Record map[string]interface{}

// Source extracts data updated after the provided timestamp.
type Source interface {
	FetchData(ctx context.Context, since time.Time) ([]Record, error)
}

// Destination loads the intermediate records into the target system.
type Destination interface {
	SaveData(ctx context.Context, records []Record) error
}

// State tracks our high-water mark for the polling query.
type State struct {
	LastTimestamp time.Time `json:"last_timestamp"`
}

// Engine orchestrates the continuous replication process.
type Engine struct {
	source    Source
	dest      Destination
	interval  time.Duration
	stateFile string
	mu        sync.Mutex
	isRunning bool
}

// NewEngine constructs our replication engine.
func NewEngine(s Source, d Destination, intervalSecs int, stateFile string) *Engine {
	return &Engine{
		source:    s,
		dest:      d,
		interval:  time.Duration(intervalSecs) * time.Second,
		stateFile: stateFile,
	}
}

// Run starts the continuous background polling loop.
func (e *Engine) Run(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	log.Printf("Starting replication engine (polling every %v)...", e.interval)

	// Execute immediately on startup before waiting for the first tick
	e.process(ctx)

	for {
		select {
		case <-ctx.Done(): // Captures shutdown signals (like CTRL+C)
			log.Println("Shutting down replication engine...")
			return
		case <-ticker.C: // Triggers every X seconds
			e.process(ctx)
		}
	}
}

// process handles a single replication cycle safely.
func (e *Engine) process(ctx context.Context) {
	// 1. Concurrency Lock: Skip if the previous cycle is still running.
	e.mu.Lock()
	if e.isRunning {
		e.mu.Unlock()
		log.Println("Previous cycle still running, skipping this interval...")
		return
	}
	e.isRunning = true
	e.mu.Unlock()

	// Ensure the lock is released when this function finishes
	defer func() {
		e.mu.Lock()
		e.isRunning = false
		e.mu.Unlock()
	}()

	// 2. Read State
	state, err := e.loadState()
	if err != nil {
		log.Printf("Error loading state: %v", err)
		return
	}

	// 3. Extract Data
	records, err := e.source.FetchData(ctx, state.LastTimestamp)

	if err != nil {
		log.Printf("Error fetching data: %v", err)
		return
	}

	if len(records) == 0 {
		return // Silently return if no new data to avoid log spam
	}

	// 4. Load Data
	if err := e.dest.SaveData(ctx, records); err != nil {
		log.Printf("Error saving data: %v", err)
		return
	}

	// 5. Update State
	// In a real application, you would extract the maximum timestamp from the 'records' array.
	// We use time.Now() here to move the watermark forward simply.
	state.LastTimestamp = time.Now().UTC()
	if err := e.saveState(state); err != nil {
		log.Printf("Error saving state: %v", err)
	}

	log.Printf("Successfully replicated %d records.", len(records))
}

// loadState reads the last saved timestamp.
func (e *Engine) loadState() (*State, error) {
	file, err := os.ReadFile(e.stateFile)
	if err != nil {
		// If file doesn't exist, return a default state
		if os.IsNotExist(err) {
			return &State{LastTimestamp: time.Time{}}, nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(file, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// saveState persists the new timestamp.
func (e *Engine) saveState(state *State) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(e.stateFile, data, 0644)
}

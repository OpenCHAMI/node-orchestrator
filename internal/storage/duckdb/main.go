package duckdb

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/rs/zerolog/log"
)

type DuckDBStorage struct {
	db                *sql.DB
	snapshotFrequency time.Duration
	snapshotPath      string
	restoreFirst      bool
	wg                sync.WaitGroup
	cancelSnapshot    context.CancelFunc
	collectionManager *nodes.CollectionManager
}

func NewDuckDBStorage(path string, options ...DuckDBStorageOption) (*DuckDBStorage, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}

	d := &DuckDBStorage{
		db:                db,
		collectionManager: nodes.NewCollectionManager(),
	}

	for _, option := range options {
		err := option.apply(d)
		if err != nil {
			log.Warn().Err(err).Msg("Error applying DuckDBStorage option")
		}
	}

	d.loadExtensions()
	d.initTables()

	ctx, cancel := context.WithCancel(context.Background())
	d.cancelSnapshot = cancel

	d.wg.Add(1)
	go d.snapshotRoutine(ctx)

	go d.handleShutdown()

	return d, nil
}

func (d *DuckDBStorage) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS compute_nodes (id UUID PRIMARY KEY, added TIMESTAMP DEFAULT CURRENT_TIMESTAMP, xname TEXT UNIQUE, data JSON)`,
		`CREATE TABLE IF NOT EXISTS bmcs (id UUID PRIMARY KEY, xname TEXT UNIQUE, added TIMESTAMP DEFAULT CURRENT_TIMESTAMP, data JSON)`,
		`CREATE TABLE IF NOT EXISTS collections (id UUID PRIMARY KEY, name TEXT UNIQUE, data JSON, nodes JSON)`,
		`CREATE INDEX IF NOT EXISTS idx_collections_nodes ON collections USING GIN (nodes)`,
	}
	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (d *DuckDBStorage) Close() error {
	return d.db.Close()
}

func (d *DuckDBStorage) handleShutdown() {
	// Create a channel to listen for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Create a context with a timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt to take a final snapshot
	log.Info().Msg("Taking final snapshot before shutdown")
	if err := d.SnapshotParquet(ctx, d.snapshotPath); err != nil {
		log.Error().Err(err).Msg("Error taking final snapshot")
	}

	// Stop the snapshot routine
	log.Info().Msg("Stopping snapshot routine")
	d.cancelSnapshot()

	// Wait for all goroutines to finish with the context's timeout
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info().Msg("All goroutines finished cleanly")
	case <-ctx.Done():
		log.Warn().Msg("Timeout waiting for goroutines to finish")
	}

	// Close the database connection
	log.Info().Msg("Closing database connection")
	if err := d.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing database connection")
	}

	log.Info().Msg("Shutdown complete")
}

func (d *DuckDBStorage) initializeDatabase() error {
	if err := d.loadExtensions(); err != nil {
		return err
	}
	return d.initTables()
}

func (d *DuckDBStorage) loadExtensions() error {
	_, err := d.db.Exec("SET autoinstall_known_extensions=1;INSTALL json;LOAD json;INSTALL parquet;LOAD parquet")
	if err != nil {
		log.Error().Err(err).Msg("Failed to load DuckDB extensions")
	}
	return err
}

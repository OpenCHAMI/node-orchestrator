package eventlogger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/sirupsen/logrus"
)

const (
	defaultBaseDir           = "/logs"
	defaultWriteInterval     = time.Hour
	defaultCleanupInterval   = 2 * time.Hour
	defaultRetainInDB        = true
	defaultDuckDBPath        = ":memory:"
	defaultPopulateFromFiles = false
)

type EventLoggerConfig struct {
	BaseDir           string
	WriteInterval     time.Duration
	CleanupInterval   time.Duration
	RetainInDB        bool
	DuckDBPath        string
	PopulateFromFiles bool
}

type EventLogger struct {
	db           *sql.DB
	log          *logrus.Logger
	config       EventLoggerConfig
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

var (
	globalLogger *EventLogger
	once         sync.Once
)

func NewEventLogger(config EventLoggerConfig) (*EventLogger, error) {
	var db *sql.DB
	var err error

	if config.DuckDBPath == ":memory:" {
		db, err = sql.Open("duckdb", "")
	} else {
		db, err = sql.Open("duckdb", config.DuckDBPath)
	}
	if err != nil {
		return nil, err
	}

	// Initialize DuckDB table
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS events (
		timestamp TIMESTAMP,
		event_type STRING,
		event_data JSON
	)`)
	if err != nil {
		return nil, err
	}

	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetOutput(os.Stdout)

	el := &EventLogger{
		db:           db,
		log:          log,
		config:       config,
		shutdownChan: make(chan struct{}),
	}

	if config.PopulateFromFiles {
		if err := el.populateDBFromFiles(); err != nil {
			return nil, err
		}
	}

	return el, nil
}

func Initialize() error {
	return InitializeWithConfig(EventLoggerConfig{
		BaseDir:           defaultBaseDir,
		WriteInterval:     defaultWriteInterval,
		CleanupInterval:   defaultCleanupInterval,
		RetainInDB:        defaultRetainInDB,
		DuckDBPath:        defaultDuckDBPath,
		PopulateFromFiles: defaultPopulateFromFiles,
	})
}

func InitializeWithConfig(config EventLoggerConfig) error {
	var err error
	once.Do(func() {
		globalLogger, err = NewEventLogger(config)
	})
	return err
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return globalLogger.WithFields(fields)
}

func (el *EventLogger) LogEvent(eventType string, eventData map[string]interface{}) {
	timestamp := time.Now().Format(time.RFC3339)

	// Log to stdout
	el.log.WithFields(logrus.Fields{
		"event":      eventType,
		"timestamp":  timestamp,
		"event_data": eventData,
	}).Info("Event logged")

	// Insert into DuckDB
	eventDataJSON, _ := json.Marshal(eventData)
	_, err := el.db.Exec(`INSERT INTO events (timestamp, event_type, event_data) VALUES (?, ?, ?)`,
		timestamp, eventType, string(eventDataJSON))
	if err != nil {
		el.log.WithError(err).Error("Failed to insert event into DuckDB")
	}
}

// Implementing WithFields method to support structured logging
func (el *EventLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	entry := el.log.WithFields(fields)
	el.LogEvent("event", fields)
	return entry
}

func (el *EventLogger) populateDBFromFiles() error {
	files, err := filepath.Glob(filepath.Join(el.config.BaseDir, "*", "*", "*", "*", "*", "part-*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		err := el.loadFileIntoDB(file)
		if err != nil {
			el.log.WithError(err).Errorf("Failed to load file %s into DuckDB", file)
		}
	}

	return nil
}

func (el *EventLogger) loadFileIntoDB(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for decoder.More() {
		var event map[string]interface{}
		if err := decoder.Decode(&event); err != nil {
			return err
		}

		timestamp := event["timestamp"].(string)
		eventType := event["event_type"].(string)
		eventData, _ := json.Marshal(event["event_data"])
		_, err = el.db.Exec(`INSERT INTO events (timestamp, event_type, event_data) VALUES (?, ?, ?)`,
			timestamp, eventType, string(eventData))
		if err != nil {
			return err
		}
	}

	return nil
}

func (el *EventLogger) FlushEvents() {
	el.wg.Add(1)
	defer el.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	rows, err := el.db.QueryContext(ctx, `SELECT timestamp, event_data FROM events`)
	if err != nil {
		el.log.WithError(err).Error("Failed to query events from DuckDB")
		return
	}
	defer rows.Close()

	eventFiles := make(map[string]*os.File)

	for rows.Next() {
		var timestamp, eventData string
		if err := rows.Scan(&timestamp, &eventData); err != nil {
			el.log.WithError(err).Error("Failed to scan event row")
			continue
		}

		t, _ := time.Parse(time.RFC3339, timestamp)
		year, month, day, hour := t.Year(), t.Month(), t.Day(), t.Hour()
		dir := fmt.Sprintf("%s/year=%d/month=%02d/day=%02d/hour=%02d",
			el.config.BaseDir, year, month, day, hour)

		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			el.log.WithError(err).Error("Failed to create directory")
			continue
		}

		filePath := fmt.Sprintf("%s/part-00000.json", dir)

		file, ok := eventFiles[filePath]
		if !ok {
			file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				el.log.WithError(err).Error("Failed to open event file")
				continue
			}
			eventFiles[filePath] = file
		}

		if _, err := file.WriteString(eventData + "\n"); err != nil {
			el.log.WithError(err).Error("Failed to write event to file")
		}

	}

	for _, file := range eventFiles {
		file.Close()
	}

	if !el.config.RetainInDB {
		_, err = el.db.ExecContext(ctx, `DELETE FROM events`)
		if err != nil {
			el.log.WithError(err).Error("Failed to clear events from DuckDB")
		}
	}
}

func (el *EventLogger) CleanupEvents() {
	el.wg.Add(1)
	defer el.wg.Done()

	if el.config.RetainInDB {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := el.db.ExecContext(ctx, `DELETE FROM events`)
	if err != nil {
		el.log.WithError(err).Error("Failed to clear events from DuckDB")
	}
}

func (el *EventLogger) StartPeriodicFlush() {
	writeTicker := time.NewTicker(el.config.WriteInterval)
	cleanupTicker := time.NewTicker(el.config.CleanupInterval)
	go func() {
		for {
			select {
			case <-writeTicker.C:
				el.FlushEvents()
			case <-cleanupTicker.C:
				el.CleanupEvents()
			case <-el.shutdownChan:
				el.FlushEvents()
				writeTicker.Stop()
				cleanupTicker.Stop()
				close(el.shutdownChan)
				return
			}
		}
	}()
}

func (el *EventLogger) Stop() {
	close(el.shutdownChan)
	el.wg.Wait()
}

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/openchami/node-orchestrator/internal/storage"
	"github.com/openchami/node-orchestrator/internal/storage/duckdb"
	openchami_middleware "github.com/openchami/node-orchestrator/pkg/middleware"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	serveCmd          = flag.NewFlagSet("serve", flag.ExitOnError)
	schemaCmd         = flag.NewFlagSet("schemas", flag.ExitOnError)
	snapshotPath      = serveCmd.String("dir", "snapshots/", "directory to store snapshots")
	schemaPath        = schemaCmd.String("dir", "schemas/", "directory to store JSON schemas")
	snapshotFreq      = serveCmd.Duration("snapshot-freq", 60*time.Minute, "frequency to take snapshots. 0 disables snapshots")
	snapshotDirCreate = serveCmd.Bool("snapshot-dir", true, "create snapshot directory if it doesn't exist")
	initTables        = serveCmd.Bool("init-tables", false, "initialize tables in the database")
	restoreSnapshot   = serveCmd.Bool("restore", true, "restore from snapshot on startup")
)

type Config struct {
	ListenAddr string
	BMCSubnet  net.IPNet // BMCSubnet is the subnet for BMCs

}

type App struct {
	Storage storage.NodeStorage
	Router  *chi.Mux
}

func main() {

	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if len(os.Args) < 2 {
		fmt.Println("expected 'serve' or 'schemas' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		serveCmd.Parse(os.Args[2:])
		serveAPI(logger)
	case "schemas":
		schemaCmd.Parse(os.Args[2:])
		generateAndWriteSchemas(*schemaPath)
	default:
		fmt.Println("expected 'serve' or 'schemas' subcommands")
		os.Exit(1)
	}
}

func serveAPI(logger zerolog.Logger) {
	// Create a new token authenticator
	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil, jwt.WithAcceptableSkew(30*time.Second))
	// Create a new chi router
	r := chi.NewRouter()
	// Add middleware to the router
	r.Use(middleware.RequestID)
	r.Use(openchami_middleware.OpenCHAMILogger(logger))
	r.Use(middleware.Recoverer)

	var authMiddleware = []func(http.Handler) http.Handler{
		jwtauth.Verifier(tokenAuth),
		openchami_middleware.AuthenticatorWithRequiredClaims(tokenAuth, []string{"sub", "iss", "aud"}),
	}

	// Initialize the storage backend options
	var options []duckdb.DuckDBStorageOption
	if serveCmd.Parsed() {
		if *initTables {
			options = append(options, duckdb.WithInitTables(*initTables))
		}
		if *snapshotPath != "" {
			log.Info().Msg("Adding the storage option to specify a snapshot path")
			options = append(options, duckdb.WithSnapshotPath(*snapshotPath))
			if *snapshotDirCreate {
				log.Info().Msg("Adding the storage option to create the snapshot directory if it doesn't exist")
				options = append(options, duckdb.WithCreateSnapshotDir(*snapshotDirCreate))
			}
			if *snapshotFreq > time.Duration(0) {
				log.Info().Msg("Adding the storage option to snapshot regularly")
				options = append(options, duckdb.WithSnapshotFrequency(*snapshotFreq))
			}
			if *restoreSnapshot {
				log.Info().Msg("Adding the storage option to restore from snapshot on startup")
				options = append(options, duckdb.WithRestore(*snapshotPath))
			}
		}
	}

	myStorage, err := duckdb.NewDuckDBStorage("data.db", options...)
	if err != nil {
		if err.Error() == "no snapshot found" {
			log.Warn().Msg("No snapshot found, starting with empty database")
		} else {
			log.Fatal().Err(err).Msg("Error creating storage")
		}
	}

	r.Mount("/inventory", NodeRoutes(myStorage, authMiddleware))

	// CSM Routes
	r.Mount("/smd", SMDComponentRoutes(myStorage, authMiddleware))

	log.Info().Msg("Starting server on :8080")
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("[%s]: '%s' has %d middlewares\n", method, route, len(middlewares))
		return nil
	})

	// Set up signal handling
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start the HTTP server
	go func() {
		if err := http.ListenAndServe(":8080", r); err != nil {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	// Wait for a signal
	sig := <-quit
	log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")

	// Create a context with a timeout for the shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call the storage shutdown method
	myStorage.Shutdown(ctx)
}

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/openchami/node-orchestrator/internal/storage"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/google/uuid"
)

var (
	serveCmd     = flag.NewFlagSet("serve", flag.ExitOnError)
	schemaCmd    = flag.NewFlagSet("schemas", flag.ExitOnError)
	snapshotCmd  = flag.NewFlagSet("snapshot", flag.ExitOnError)
	snapshotPath = snapshotCmd.String("dir", "snapshots/", "directory to store snapshots")
	schemaPath   = schemaCmd.String("dir", "schemas/", "directory to store JSON schemas")
)

type Config struct {
	ListenAddr string
	BMCSubnet  net.IPNet // BMCSubnet is the subnet for BMCs

}

type App struct {
	Storage storage.Storage
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
	case "snapshot":
		snapshotCmd.Parse(os.Args[2:])
		snapshot()
	case "restore":
		snapshotCmd.Parse(os.Args[2:])
		restore()
	default:
		fmt.Println("expected 'serve', 'snapshot', or 'schemas' subcommands")
		os.Exit(1)
	}
}

func serveAPI(logger zerolog.Logger) {
	// Create a new token authenticator
	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil, jwt.WithAcceptableSkew(30*time.Second))
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(OpenCHAMILogger(logger))
	r.Use(middleware.Recoverer)
	myStorage, err := storage.NewDuckDBStorage("data.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating storage")
	}

	app := &App{
		Storage: myStorage,
		Router:  r,
	}

	manager := nodes.NewCollectionManager()
	manager.AddConstraint(nodes.DefaultType, &nodes.MutualExclusivityConstraint{ExistingNodes: make(map[xnames.NodeXname]uuid.UUID)})

	// Protected routes
	r.Group(func(r chi.Router) {
		// Seek, verify and validate JWT tokens
		r.Use(jwtauth.Verifier(tokenAuth))

		// Handle valid / invalid tokens.
		r.Use(AuthenticatorWithRequiredClaims(tokenAuth, []string{"sub", "iss", "aud"}))
		r.Put("/ComputeNode/{nodeID}", updateNode(app.Storage))
		r.Post("/ComputeNode", postNode(app.Storage))
		r.Delete("/ComputeNode/{nodeID}", deleteNode(app.Storage))

		r.Post("/nodes", postNode(app.Storage))
		r.Put("/nodes/{nodeID}", updateNode(app.Storage))
		r.Delete("/nodes/{nodeID}", deleteNode(app.Storage))

		r.Post("/bmc", postBMC(app.Storage))
		r.Put("/bmc/{bmcID}", updateBMC(app.Storage))
		r.Delete("/bmc/{bmcID}", deleteBMC(app.Storage))

		r.Post("/NodeCollection", createCollection(manager))
		r.Put("/NodeCollection/{identifier}", updateCollection(manager))
		r.Delete("/NodeCollection/{identifier}", deleteCollection(manager))

	})

	// Public routes

	r.Get("/ComputeNode/{nodeID}", getNode(app.Storage))
	r.Get("/ComputeNode", searchNodes(app.Storage))
	r.Get("/nodes/{nodeID}", getNode(app.Storage))
	r.Get("/nodes", searchNodes(app.Storage))
	r.Get("/bmc/{bmcID}", getBMC(app.Storage))
	r.Get("/NodeCollection/{identifier}", getCollection(manager))

	log.Info().Msg("Starting server on :8080")
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("[%s]: '%s' has %d middlewares\n", method, route, len(middlewares))
		return nil
	})

	log.Fatal().Err(http.ListenAndServe(":8080", r))
}

func snapshot() {
	log.Info().Msg("Taking snapshot")
	myStorage, err := storage.NewDuckDBStorage("data.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating storage")
	}
	err = myStorage.SnapshotParquet(*snapshotPath)
	if err != nil {
		log.Fatal()
	}
}

func restore() {
	log.Info().Msg("Restoring snapshot")
	myStorage, err := storage.NewDuckDBStorageForRestore("data.db")
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating storage")
	}

	// Find the most recent snapshot directory
	snapshotDir, err := findMostRecentSnapshotDir(*snapshotPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Error finding snapshot directory")
	}

	err = myStorage.RestoreParquet(snapshotDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Error restoring snapshot")
	}
}

// findMostRecentSnapshotDir finds the most recent directory under the given path
func findMostRecentSnapshotDir(path string) (string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return "", err
	}

	var dirs []os.FileInfo
	for _, file := range files {
		if file.IsDir() {
			dirs = append(dirs, file)
		}
	}

	if len(dirs) == 0 {
		return "", fmt.Errorf("no snapshot directories found")
	}

	// Sort directories by name (assuming they are named by date)
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Name() > dirs[j].Name() // descending order
	})

	// Return the most recent directory
	mostRecentDir := filepath.Join(path, dirs[0].Name())
	return mostRecentDir, nil
}

func AuthenticatorWithRequiredClaims(ja *jwtauth.JWTAuth, requiredClaims []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, claims, err := jwtauth.FromContext(r.Context())

			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if token == nil || jwt.Validate(token, ja.ValidateOptions()...) != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}

			for _, claim := range requiredClaims {
				if _, ok := claims[claim]; !ok {
					err := fmt.Errorf("missing required claim %s", claim)
					log.Error().Err(err).Msg("Missing required claim")
					http.Error(w, "missing required claim", http.StatusUnauthorized)
					return
				}
			}

			// Token is authenticated and all required claims are present, pass it through
			next.ServeHTTP(w, r)
		})
	}
}

const LoggerKey = "logger"

// OpenCHAMILogger is a chi middleware that adds a sublogger to the context.
func OpenCHAMILogger(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sublogger := logger.With().
				Str("request_id", middleware.GetReqID(r.Context())).
				Str("request_uri", r.RequestURI).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Str("method", r.Method).
				Logger()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			ctx := context.WithValue(r.Context(), LoggerKey, &sublogger)

			// Use the modified context with the sublogger
			r = r.WithContext(ctx)

			defer func() {
				duration := time.Since(start)
				// Extract the sublogger from the context again
				sublogger := r.Context().Value(LoggerKey).(*zerolog.Logger)
				sublogger.Info().
					Str("status", http.StatusText(ww.Status())).
					Int("status_code", ww.Status()).
					Int64("bytes_in", r.ContentLength).
					Int("bytes_out", ww.BytesWritten()).
					Dur("duration", duration).
					Msg("Request")
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

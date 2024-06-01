package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/openchami/node-orchestrator/internal/storage"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
)

var (
	serveCmd   = flag.NewFlagSet("serve", flag.ExitOnError)
	schemaCmd  = flag.NewFlagSet("schemas", flag.ExitOnError)
	schemaPath = schemaCmd.String("dir", "schemas/", "directory to store JSON schemas")
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

	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	if len(os.Args) < 2 {
		fmt.Println("expected 'serve' or 'schemas' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		serveCmd.Parse(os.Args[2:])
		serveAPI()
	case "schemas":
		schemaCmd.Parse(os.Args[2:])
		generateAndWriteSchemas(*schemaPath)
	default:
		fmt.Println("expected 'serve' or 'schemas' subcommands")
		os.Exit(1)
	}
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
					log.WithError(err).Error("Missing required claim")
					http.Error(w, "missing required claim", http.StatusUnauthorized)
					return
				}
			}

			// Token is authenticated and all required claims are present, pass it through
			next.ServeHTTP(w, r)
		})
	}
}

func serveAPI() {
	// Create a new token authenticator
	tokenAuth := jwtauth.New("HS256", []byte("secret"), nil, jwt.WithAcceptableSkew(30*time.Second))
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	myStorage, err := storage.NewDuckDBStorage("data.db")
	if err != nil {
		log.WithError(err).Fatal("Error creating storage")
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

	log.Info("Starting server on :8080")
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("[%s]: '%s' has %d middlewares\n", method, route, len(middlewares))
		return nil
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}

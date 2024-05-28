package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/lestrrat-go/jwx/v2/jwt"

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
	Storage Storage
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

	app := &App{
		Storage: NewInMemoryStorage(),
		Router:  r,
	}

	manager := NewCollectionManager()
	manager.AddConstraint(DefaultType, &MutualExclusivityConstraint{existingNodes: make(map[NodeXname]uuid.UUID)})

	// Protected routes
	r.Group(func(r chi.Router) {
		// Seek, verify and validate JWT tokens
		r.Use(jwtauth.Verifier(tokenAuth))

		// Handle valid / invalid tokens.
		r.Use(AuthenticatorWithRequiredClaims(tokenAuth, []string{"sub", "iss", "aud"}))
		r.Put("/ComputeNode/{nodeID}", app.updateNode)
		r.Post("/ComputeNode", app.postNode)
		r.Delete("/ComputeNode/{nodeID}", app.deleteNode)

		r.Post("/nodes", app.postNode)
		r.Put("/nodes/{nodeID}", app.updateNode)
		r.Delete("/nodes/{nodeID}", app.deleteNode)

		r.Post("/bmc", app.postBMC)
		r.Put("/bmc/{bmcID}", app.updateBMC)
		r.Delete("/bmc/{bmcID}", app.deleteBMC)

		r.Post("/NodeCollection", createCollection(manager))
		r.Put("/NodeCollection/{identifier}", updateCollection(manager))
		r.Delete("/NodeCollection/{identifier}", deleteCollection(manager))

	})

	// Public routes

	r.Get("/ComputeNode/{nodeID}", app.getNode)
	r.Get("/nodes/{nodeID}", app.getNode)
	r.Get("/bmc/{bmcID}", app.getBMC)
	r.Get("/NodeCollection/{identifier}", getCollection(manager))

	log.Info("Starting server on :8080")
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("[%s]: '%s' has %d middlewares\n", method, route, len(middlewares))
		return nil
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}

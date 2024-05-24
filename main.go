package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
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

func serveAPI() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	app := &App{
		Storage: NewInMemoryStorage(),
		Router:  r,
	}

	r.Post("/ComputeNode", app.postNode)
	r.Get("/ComputeNode/{nodeID}", app.getNode)
	r.Put("/ComputeNode/{nodeID}", app.updateNode)
	r.Delete("/ComputeNode/{nodeID}", app.deleteNode)

	r.Post("/nodes", app.postNode)
	r.Get("/nodes/{nodeID}", app.getNode)
	r.Put("/nodes/{nodeID}", app.updateNode)
	r.Delete("/nodes/{nodeID}", app.deleteNode)

	r.Post("/bmc", app.postBMC)
	r.Get("/bmc/{bmcID}", app.getBMC)
	r.Put("/bmc/{bmcID}", app.updateBMC)
	r.Delete("/bmc/{bmcID}", app.deleteBMC)

	manager := NewCollectionManager()
	manager.AddConstraint(DefaultType, &MutualExclusivityConstraint{existingNodes: make(map[NodeXname]uuid.UUID)})

	r.Route("/NodeCollection", func(r chi.Router) {
		r.Post("/", createCollection(manager))
		r.Get("/{identifier}", getCollection(manager))
		r.Put("/{identifier}", updateCollection(manager))
		r.Delete("/{identifier}", deleteCollection(manager))
	})

	log.Printf("Starting server on :8080")
	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("[%s]: '%s' has %d middlewares\n", method, route, len(middlewares))
		return nil
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}

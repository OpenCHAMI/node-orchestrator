package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
)

var (
	serveCmd   = flag.NewFlagSet("serve", flag.ExitOnError)
	schemaCmd  = flag.NewFlagSet("schemas", flag.ExitOnError)
	schemaPath = schemaCmd.String("dir", "schemas/", "directory to store JSON schemas")
)

type Storage interface {
	SaveComputeNode(nodeID uuid.UUID, node ComputeNode) error
	GetComputeNode(nodeID uuid.UUID) (ComputeNode, error)
	UpdateComputeNode(nodeID uuid.UUID, node ComputeNode) error
	DeleteComputeNode(nodeID uuid.UUID) error

	LookupComputeNodeByXName(xname string) (ComputeNode, error)
	LookupComputeNodeByMACAddress(mac string) (ComputeNode, error)

	SaveBMC(bmcID uuid.UUID, bmc BMC) error
	GetBMC(bmcID uuid.UUID) (BMC, error)
	UpdateBMC(bmcID uuid.UUID, bmc BMC) error
	DeleteBMC(bmcID uuid.UUID) error

	LookupBMCByXName(xname string) (BMC, error)
	LookupBMCByMACAddress(mac string) (BMC, error)
}

type App struct {
	Storage Storage
	Router  *chi.Mux
}

func generateAndWriteSchemas(path string) {
	schemas := map[string]interface{}{
		"ComputeNode.json":      &ComputeNode{},
		"NetworkInterface.json": &NetworkInterface{},
		"BMC.json":              &BMC{},
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		log.Fatalf("Failed to create schema directory: %v", err)
	}

	for filename, model := range schemas {
		schema := jsonschema.Reflect(model)
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			log.Fatalf("Failed to generate JSON schema for %v: %v", filename, err)
		}
		fullpath := filepath.Join(path, filename)
		if err := os.WriteFile(fullpath, data, 0644); err != nil {
			log.Fatalf("Failed to write JSON schema to file %v: %v", fullpath, err)
		}
		fmt.Printf("Schema written to %s\n", fullpath)
	}
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

	chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		fmt.Printf("[%s]: '%s' has %d middlewares\n", method, route, len(middlewares))
		return nil
	})

	log.Fatal(http.ListenAndServe(":8080", r))
}

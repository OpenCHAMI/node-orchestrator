package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	base "github.com/Cray-HPE/hms-base"
	"github.com/invopop/jsonschema"
)

func generateAndWriteSchemas(path string) {
	schemas := map[string]interface{}{
		"ComputeNode.json":      &ComputeNode{},
		"NetworkInterface.json": &NetworkInterface{},
		"BMC.json":              &BMC{},
		"Component.json":        &base.Component{},
		"NodeCollection.json":   &NodeCollection{},
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

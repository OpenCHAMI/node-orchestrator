package main

import (
	"encoding/json"
	"fmt"

	"os"
	"path/filepath"

	base "github.com/Cray-HPE/hms-base"
	"github.com/invopop/jsonschema"
	nodes "github.com/openchami/node-orchestrator/pkg/nodes"
	log "github.com/sirupsen/logrus"
)

func generateAndWriteSchemas(path string) {
	schemas := map[string]interface{}{
		"ComputeNode.json":      &nodes.ComputeNode{},
		"NetworkInterface.json": &nodes.NetworkInterface{},
		"BMC.json":              &nodes.BMC{},
		"Component.json":        &base.Component{},
		"NodeCollection.json":   &nodes.NodeCollection{},
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		log.WithError(err).Error("Failed to create schema directory")
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

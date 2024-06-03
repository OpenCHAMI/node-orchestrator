package main

import (
	"encoding/json"

	"os"
	"path/filepath"

	base "github.com/Cray-HPE/hms-base"
	"github.com/invopop/jsonschema"
	nodes "github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/rs/zerolog/log"
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
		log.Fatal().Err(err).Str("path", path).Msg("Failed to create schema directory")
	}

	for filename, model := range schemas {
		schema := jsonschema.Reflect(model)
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			log.Fatal().Err(err).Str("filename", filename).Msg("Failed to generate JSON schema")
		}
		fullpath := filepath.Join(path, filename)
		if err := os.WriteFile(fullpath, data, 0644); err != nil {
			log.Fatal().Err(err).Str("filename", filename).Msg("Failed to write JSON schema to file")
		}
		log.Info().Str("fullpath", fullpath).Msg("Schema written")
	}
}

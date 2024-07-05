package duckdb

import (
	"encoding/json"

	"github.com/openchami/node-orchestrator/internal/storage"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/rs/zerolog/log"
)

func (d *DuckDBStorage) SearchComputeNodes(opts ...storage.NodeSearchOption) ([]nodes.ComputeNode, error) {
	options := &storage.NodeSearchOptions{}
	for _, opt := range opts {
		opt(options)
	}

	var queryStrings []string
	var queryArgs []interface{}

	if options.XName != "" {
		queryStrings = append(queryStrings, "json_extract(data, '$.xname')::text = ?")
		queryArgs = append(queryArgs, `"`+options.XName+`"`)
	}
	if options.Hostname != "" {
		queryStrings = append(queryStrings, "json_extract(data, '$.hostname')::text = ?")
		queryArgs = append(queryArgs, `"`+options.Hostname+`"`)
	}
	if options.Arch != "" {
		queryStrings = append(queryStrings, "json_extract(data, '$.arch')::text = ?")
		queryArgs = append(queryArgs, `"`+options.Arch+`"`)
	}
	if options.BootMAC != "" {
		queryStrings = append(queryStrings, "json_extract(data, '$.boot_mac')::text = ?")
		queryArgs = append(queryArgs, `"`+options.BootMAC+`"`)
	}
	if options.BMCMAC != "" {
		queryStrings = append(queryStrings, "json_extract(data, '$.bmc.mac_address')::text = ?")
		queryArgs = append(queryArgs, `"`+options.BMCMAC+`"`)
	}

	if options.MissingXName {
		queryStrings = append(queryStrings, "json_extract(data, '$.xname') IS NULL")
	}
	if options.MissingHostname {
		queryStrings = append(queryStrings, "json_extract(data, '$.hostname') IS NULL")
	}
	if options.MissingArch {
		queryStrings = append(queryStrings, "json_extract(data, '$.arch') IS NULL")
	}
	if options.MissingBootMAC {
		queryStrings = append(queryStrings, "json_extract(data, '$.boot_mac') IS NULL")
	}
	if options.MissingBMCMAC {
		queryStrings = append(queryStrings, "json_extract(data, '$.bmc.mac_address') IS NULL")
	}
	if options.MissingIPV4 {
		queryStrings = append(queryStrings, "json_extract(data, '$.boot_ipv4_address') IS NULL")
	}
	if options.MissingIPV6 {
		queryStrings = append(queryStrings, "json_extract(data, '$.boot_ipv4_address') IS NULL")
	}

	query := buildQuery("AND", queryStrings...)

	rows, err := d.db.Query(query, queryArgs...)
	if err != nil {
		log.Error().Err(err).Msg("Error querying DuckDB for ComputeNodes")
		return nil, err
	}
	defer rows.Close()

	var foundNodes []nodes.ComputeNode
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var node nodes.ComputeNode
		if err := json.Unmarshal([]byte(data), &node); err != nil {
			return nil, err
		}
		foundNodes = append(foundNodes, node)
	}

	log.Debug().Str("query", query).Interface("args", queryArgs).Int("count", len(foundNodes)).Msg("DuckDB ComputeNode search complete")
	return foundNodes, nil
}

// buildQuery builds a SQL query for searching compute nodes
func buildQuery(condition string, fields ...string) string {
	query := "SELECT data FROM compute_nodes WHERE 1=1"
	for _, field := range fields {
		query += " " + condition + " " + field
	}
	return query
}

package duckdb

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/rs/zerolog/log"
)

func (d *DuckDBStorage) SaveComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error {
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT INTO compute_nodes (id, xname, data) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data`, nodeID, node.XName.Value, string(data))
	return err
}

func (d *DuckDBStorage) GetComputeNode(nodeID uuid.UUID) (nodes.ComputeNode, error) {
	var data string
	err := d.db.QueryRow(`SELECT data FROM compute_nodes WHERE id = ?`, nodeID).Scan(&data)
	if err != nil {
		return nodes.ComputeNode{}, err
	}
	var node nodes.ComputeNode
	err = json.Unmarshal([]byte(data), &node)
	return node, err
}

func (d *DuckDBStorage) UpdateComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error {
	return d.SaveComputeNode(nodeID, node)
}

func (d *DuckDBStorage) DeleteComputeNode(nodeID uuid.UUID) error {
	_, err := d.db.Exec(`DELETE FROM compute_nodes WHERE id = ?`, nodeID)
	return err
}

func (d *DuckDBStorage) LookupComputeNodeByXName(xname string) (nodes.ComputeNode, error) {
	var data string
	err := d.db.QueryRow(`SELECT data FROM compute_nodes WHERE json_extract(data, '$.xname') = ?`, xname).Scan(&data)
	if err != nil {
		return nodes.ComputeNode{}, err
	}
	var node nodes.ComputeNode
	err = json.Unmarshal([]byte(data), &node)
	return node, err
}

func (d *DuckDBStorage) LookupComputeNodeByMACAddress(mac string) (nodes.ComputeNode, error) {
	var data string
	err := d.db.QueryRow(`SELECT data FROM compute_nodes WHERE json_extract(data, '$.boot_mac') = ?`, mac).Scan(&data)
	if err != nil {
		return nodes.ComputeNode{}, err
	}
	var node nodes.ComputeNode
	err = json.Unmarshal([]byte(data), &node)
	return node, err
}

func (d *DuckDBStorage) SearchComputeNodes(xname, hostname, arch, bootMAC, bmcMAC string) ([]nodes.ComputeNode, error) {
	var queryStrings []string
	var queryArgs []interface{}
	if xname != "" {
		queryStrings = append(queryStrings, "json_extract(data, '$.xname')::text = ?")
		queryArgs = append(queryArgs, `"`+xname+`"`)
	}
	if hostname != "" {
		queryStrings = append(queryStrings, " json_extract(data, '$.hostname')::text = ? ")
		queryArgs = append(queryArgs, `"`+hostname+`"`)
	}
	if arch != "" {
		queryStrings = append(queryStrings, " json_extract(data, '$.arch')::text = ? ")
		queryArgs = append(queryArgs, `"`+arch+`"`)
	}
	if bootMAC != "" {
		queryStrings = append(queryStrings, " json_extract(data, '$.boot_mac')::text = ? ")
		queryArgs = append(queryArgs, `"`+bootMAC+`"`)
	}
	if bmcMAC != "" {
		queryStrings = append(queryStrings, " json_extract(data, '$.bmc.mac_address')::text = ? ")
		queryArgs = append(queryArgs, `"`+bmcMAC+`"`)
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

func (d *DuckDBStorage) SaveBMC(bmcID uuid.UUID, bmc nodes.BMC) error {
	data, err := json.Marshal(bmc)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT INTO bmcs (id, data) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		bmcID, string(data))
	return err
}

func (d *DuckDBStorage) GetBMC(bmcID uuid.UUID) (nodes.BMC, error) {
	var data string
	err := d.db.QueryRow(`SELECT data FROM bmcs WHERE id = ?`, bmcID).Scan(&data)
	if err != nil {
		return nodes.BMC{}, err
	}
	var bmc nodes.BMC
	err = json.Unmarshal([]byte(data), &bmc)
	return bmc, err
}

func (d *DuckDBStorage) UpdateBMC(bmcID uuid.UUID, bmc nodes.BMC) error {
	return d.SaveBMC(bmcID, bmc)
}

func (d *DuckDBStorage) DeleteBMC(bmcID uuid.UUID) error {
	_, err := d.db.Exec(`DELETE FROM bmcs WHERE id = ?`, bmcID)
	return err
}

func (d *DuckDBStorage) LookupBMCByMACAddress(mac string) (nodes.BMC, error) {
	var data string
	err := d.db.QueryRow(`SELECT data FROM bmcs WHERE json_extract(data, '$.mac_address') = ?`, mac).Scan(&data)
	if err != nil {
		return nodes.BMC{}, err
	}
	var bmc nodes.BMC
	err = json.Unmarshal([]byte(data), &bmc)
	return bmc, err
}

func (d *DuckDBStorage) LookupBMCByXName(xname string) (nodes.BMC, error) {
	var data string
	err := d.db.QueryRow(`SELECT data FROM bmcs WHERE json_extract(data, '$.xname') = ?`, xname).Scan(&data)
	if err != nil {
		return nodes.BMC{}, err
	}
	var bmc nodes.BMC
	err = json.Unmarshal([]byte(data), &bmc)
	return bmc, err
}

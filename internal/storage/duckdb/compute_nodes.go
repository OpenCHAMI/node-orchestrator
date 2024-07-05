package duckdb

import (
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/nodes"
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

func initNodeTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS compute_nodes (id UUID PRIMARY KEY, added TIMESTAMP DEFAULT CURRENT_TIMESTAMP, xname TEXT UNIQUE, boot_mac TEXT UNIQUE, data JSON)`,
		`CREATE TABLE IF NOT EXISTS bmcs (id UUID PRIMARY KEY, xname TEXT UNIQUE, added TIMESTAMP DEFAULT CURRENT_TIMESTAMP, data JSON)`,
		`CREATE TABLE IF NOT EXISTS collections (id UUID PRIMARY KEY, name TEXT UNIQUE, data JSON, nodes JSON)`,
		`CREATE INDEX IF NOT EXISTS idx_collections_nodes ON collections (nodes)`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

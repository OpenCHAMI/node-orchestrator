package storage

import (
	"database/sql"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"

	"encoding/json"

	"github.com/openchami/node-orchestrator/pkg/nodes"
)

// DuckDBStorage is a storage backend that uses DuckDB

type DuckDBStorage struct {
	db *sql.DB
}

func NewDuckDBStorage(path string) (*DuckDBStorage, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}

	d := &DuckDBStorage{db: db}

	// load extensions
	d.db.Exec("INSTALL json; LOAD json")

	if err := d.initTables(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *DuckDBStorage) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS compute_nodes (
			id UUID PRIMARY KEY,
			data JSON
		)`,
		`CREATE TABLE IF NOT EXISTS bmcs (
			id UUID PRIMARY KEY,
			data JSON
		)`,
	}
	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (d *DuckDBStorage) SaveComputeNode(nodeID uuid.UUID, node nodes.ComputeNode) error {
	data, err := json.Marshal(node)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT INTO compute_nodes (id, data) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		nodeID, string(data))
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

func (d *DuckDBStorage) Close() error {
	return d.db.Close()
}

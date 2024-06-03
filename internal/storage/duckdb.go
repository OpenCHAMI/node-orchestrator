package storage

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"

	"encoding/json"

	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/rs/zerolog/log"
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

func NewDuckDBStorageForRestore(path string) (*DuckDBStorage, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}

	d := &DuckDBStorage{db: db}

	// load extensions
	d.db.Exec("INSTALL json; LOAD json")

	return d, nil
}

func (d *DuckDBStorage) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS compute_nodes (
			id UUID PRIMARY KEY,
			added TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			xname TEXT UNIQUE,
			data JSON
		)`,
		`CREATE TABLE IF NOT EXISTS bmcs (
			id UUID PRIMARY KEY,
			xname TEXT UNIQUE,
			added TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
	_, err = d.db.Exec(`INSERT INTO compute_nodes (id, xname, data) VALUES (?, ?, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data`,
		nodeID, node.XName.Value, string(data))
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

// buildQuery builds a SQL query for searching compute nodes
func buildQuery(condition string, fields ...string) string {
	query := "SELECT data FROM compute_nodes WHERE 1=1"
	for _, field := range fields {
		query += " " + condition + " " + field
	}
	return query
}

func (d *DuckDBStorage) SearchComputeNodes(xname, hostname, arch, bootMAC, bmcMAC string) ([]nodes.ComputeNode, error) {
	// Examine each parameter and build a query that includes it if it is not empty
	var queryStrings []string
	var queryArgs []interface{} // We know these are all strings, but we need pass them as []interface{} to db.Query
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
	// Log the query and number of rows returned
	log.Debug().
		Str("query", query).
		Interface("args", queryArgs).
		Int("count", len(foundNodes)).
		Msg("DuckDB ComputeNode search complete")
	return foundNodes, nil
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

func (d *DuckDBStorage) SnapshotParquet(path string) error {
	// Ensure the path is escaped properly
	escapedPath := strings.ReplaceAll(path, "'", "''")
	// Add a trailing slash if it is missing
	if !strings.HasSuffix(escapedPath, "/") {
		escapedPath += "/"
	}
	// Add a date and time to the path
	escapedPath += time.Now().Format("2006-01-02T15-04-05")
	if !strings.HasSuffix(escapedPath, "/") {
		escapedPath += "/"
	}
	// Ensure the directory exists
	os.MkdirAll(escapedPath, 0755)

	// Construct the SQL statement
	sql := fmt.Sprintf(`INSTALL parquet;
	LOAD parquet;
	EXPORT DATABASE '%s' (FORMAT PARQUET);`, escapedPath)

	// Execute the SQL statement
	_, err := d.db.Exec(sql)
	if err != nil {
		log.Error().Err(err).Msg("Error exporting DuckDB database to Parquet format")
		return err
	}
	log.Info().
		Str("path", escapedPath).
		Msg("SnapshotParquet")

	return nil
}

func (d *DuckDBStorage) RestoreParquet(path string) error {
	// Read and execute schema.sql to set up the database schema
	schemaFile := filepath.Join(path, "schema.sql")
	if err := d.executeSQLFile(schemaFile); err != nil {
		return fmt.Errorf("error executing schema.sql: %w", err)
	}
	log.Info().Str("file", schemaFile).Msg("Executed schema.sql")

	// Read and execute load.sql to load Parquet files
	loadFile := filepath.Join(path, "load.sql")
	if err := d.executeSQLFile(loadFile); err != nil {
		return fmt.Errorf("error executing load.sql: %w", err)
	}
	log.Info().Str("file", loadFile).Msg("Executed load.sql")

	return nil
}

func (d *DuckDBStorage) executeSQLFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var sb strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		sb.WriteString(line)
		if strings.HasSuffix(strings.TrimSpace(line), ";") {
			_, err := d.db.Exec(sb.String())
			if err != nil {
				return err
			}
			sb.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

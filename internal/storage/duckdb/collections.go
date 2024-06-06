package duckdb

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
)

func (d *DuckDBStorage) SaveCollection(collection *nodes.NodeCollection) error {
	if err := d.collectionManager.CreateCollection(collection); err != nil {
		return err
	}

	data, err := json.Marshal(collection)
	if err != nil {
		return err
	}
	nodesData, err := json.Marshal(collection.Nodes)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`INSERT INTO collections (id, name, data, nodes) VALUES (?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data, nodes = excluded.nodes`, collection.ID, collection.Name, string(data), string(nodesData))
	return err
}

func (d *DuckDBStorage) GetCollection(id uuid.UUID) (*nodes.NodeCollection, error) {
	var data string
	err := d.db.QueryRow(`SELECT data FROM collections WHERE id = ?`, id).Scan(&data)
	if err != nil {
		return nil, err
	}
	var collection nodes.NodeCollection
	err = json.Unmarshal([]byte(data), &collection)
	return &collection, err
}

func (d *DuckDBStorage) UpdateCollection(collection *nodes.NodeCollection) error {
	if err := d.collectionManager.UpdateCollection(collection); err != nil {
		return err
	}

	data, err := json.Marshal(collection)
	if err != nil {
		return err
	}

	_, err = d.db.Exec(`UPDATE collections SET data = ? WHERE id = ?`, string(data), collection.ID)
	return err
}

func (d *DuckDBStorage) DeleteCollection(id uuid.UUID) error {
	_, err := d.GetCollection(id)
	if err != nil {
		return err
	}

	if err := d.collectionManager.DeleteCollection(id); err != nil {
		return err
	}

	_, err = d.db.Exec(`DELETE FROM collections WHERE id = ?`, id)
	return err
}

func (d *DuckDBStorage) FindCollectionsByNode(nodeID xnames.NodeXname) ([]*nodes.NodeCollection, error) {
	query := `SELECT data FROM collections WHERE json_contains(nodes, ?)`

	rows, err := d.db.Query(query, nodeID.Value)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var collections []*nodes.NodeCollection
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var collection nodes.NodeCollection
		if err := json.Unmarshal([]byte(data), &collection); err != nil {
			return nil, err
		}
		collections = append(collections, &collection)
	}
	return collections, nil
}

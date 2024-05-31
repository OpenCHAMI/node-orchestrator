package nodes

import (
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/openchami/node-orchestrator/pkg/xnames"
)

// NodeCollectionType represents the type of a collection.
type NodeCollectionType string

const (
	DefaultType   NodeCollectionType = "ad-hoc"
	TenantType    NodeCollectionType = "tenant"
	JobType       NodeCollectionType = "job"
	PartitionType NodeCollectionType = "partition"
)

// JSONSchema for NodeCollectionType to enforce enum and description.
func (NodeCollectionType) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Enum:        []interface{}{"ad-hoc", "tenant", "job", "partition"},
		Title:       "NodeCollectionType",
		Description: "The type of the collection. partition and tenant collections have constraints such that a node cannot be part of two partitions or part of two tenants.",
	}
}

// NodeCollection represents an arbitrary collection of nodes.
type NodeCollection struct {
	ID             uuid.UUID          `json:"id,omitempty" format:"uuid"`
	Owner          uuid.UUID          `json:"owner,omitempty" format:"uuid"`            // UUID of the owner of the collection
	CreatorSubject string             `json:"creator_subject,omitempty" format:"email"` // JWT subject of the creator of the collection
	Description    string             `json:"description,omitempty"`
	Name           string             `json:"name"`
	Type           NodeCollectionType `json:"type"`
	Nodes          []xnames.NodeXname `json:"nodes"`           // List of ComputeNode IDs
	Alias          string             `json:"alias,omitempty"` // Optional alias for the collection
}

func (c *NodeCollection) Bind(r *http.Request) error {
	if err := render.DecodeJSON(r.Body, &c); err != nil {
		log.WithFields(log.Fields{
			"error": fmt.Errorf("error decoding request body: %v", err),
		}).Error(fmt.Printf("Error decoding request body: %v", err))
		return err
	}
	return nil
}

// CollectionConstraint defines methods to enforce constraints on collections.
type CollectionConstraint interface {
	Validate(nodes []xnames.NodeXname) error
}

// MutualExclusivityConstraint ensures nodes are only in one collection of this type.
type MutualExclusivityConstraint struct {
	ExistingNodes map[xnames.NodeXname]uuid.UUID // Map of nodeID to collectionID
}

func (c *MutualExclusivityConstraint) Validate(nodes []xnames.NodeXname) error {
	for _, nodeID := range nodes {
		if _, exists := c.ExistingNodes[nodeID]; exists {
			return fmt.Errorf("node %s is already assigned to another collection", nodeID)
		}
	}
	return nil
}

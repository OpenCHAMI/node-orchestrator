package nodes

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

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

// String returns the string representation of NodeCollectionType.
func (n NodeCollectionType) String() string {
	return string(n)
}

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
	Nodes          []xnames.NodeXname `json:"nodes"`                     // List of ComputeNode IDs
	CloudInitData  map[string]string  `json:"cloud_init_data,omitempty"` // Optional cloud-init data for the collection.  It will be available in the payload as `group_{Name}`
}

func (c *NodeCollection) Bind(r *http.Request) error {
	if err := render.DecodeJSON(r.Body, &c); err != nil {
		log.Error().Err(err).Msg("Error binding collection")
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

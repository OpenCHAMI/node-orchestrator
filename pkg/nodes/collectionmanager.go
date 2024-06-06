package nodes

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/xnames"
)

// CollectionManager manages collections with constraints.
type CollectionManager struct {
	CollectionsByID   map[uuid.UUID]*NodeCollection
	CollectionsByName map[string]*NodeCollection
	Constraints       map[NodeCollectionType]CollectionConstraint
}

func NewCollectionManager() *CollectionManager {
	manager := &CollectionManager{
		CollectionsByID:   make(map[uuid.UUID]*NodeCollection),
		CollectionsByName: make(map[string]*NodeCollection),
		Constraints:       make(map[NodeCollectionType]CollectionConstraint),
	}
	// Add constraints for each type if needed
	manager.AddConstraint(PartitionType, &MutualExclusivityConstraint{ExistingNodes: make(map[xnames.NodeXname]uuid.UUID)})
	manager.AddConstraint(TenantType, &MutualExclusivityConstraint{ExistingNodes: make(map[xnames.NodeXname]uuid.UUID)})
	// Add other constraints as necessary
	return manager
}

func (m *CollectionManager) AddConstraint(collectionType NodeCollectionType, constraint CollectionConstraint) {
	m.Constraints[collectionType] = constraint
}

func (m *CollectionManager) CreateCollection(collection *NodeCollection) error {
	collection.ID = uuid.New() // Generate a new UUID for the collection

	if constraint, exists := m.Constraints[NodeCollectionType(collection.Type)]; exists {
		if err := constraint.Validate(collection.Nodes); err != nil {
			return err
		}
	}

	if collection.Name != "" {
		if _, exists := m.CollectionsByName[collection.Name]; exists {
			return fmt.Errorf("alias %s is already in use", collection.Name)
		}
		m.CollectionsByName[collection.Name] = collection
	}

	m.CollectionsByID[collection.ID] = collection
	if constraint, exists := m.Constraints[NodeCollectionType(collection.Type)]; exists {
		if mec, ok := constraint.(*MutualExclusivityConstraint); ok {
			for _, nodeID := range collection.Nodes {
				mec.ExistingNodes[nodeID] = collection.ID
			}
		}
	}
	return nil
}

func (m *CollectionManager) UpdateCollection(collection *NodeCollection) error {
	if constraint, exists := m.Constraints[collection.Type]; exists {
		if err := constraint.Validate(collection.Nodes); err != nil {
			return err
		}
	}

	if collection.Name != "" {
		if _, exists := m.CollectionsByName[collection.Name]; exists && m.CollectionsByName[collection.Name].ID != collection.ID {
			return fmt.Errorf("alias %s is already in use", collection.Name)
		}
		m.CollectionsByName[collection.Name] = collection
	}

	m.CollectionsByID[collection.ID] = collection
	return nil
}

func (m *CollectionManager) DeleteCollection(collectionID uuid.UUID) error {
	collection, exists := m.CollectionsByID[collectionID]
	if !exists {
		return fmt.Errorf("collection %s not found", collectionID)
	}

	if collection.Name != "" {
		delete(m.CollectionsByName, collection.Name)
	}
	delete(m.CollectionsByID, collectionID)
	return nil
}

func (m *CollectionManager) GetCollection(identifier string) (*NodeCollection, bool) {
	id, _ := uuid.Parse(identifier)
	if collection, exists := m.CollectionsByID[id]; exists {
		return collection, true
	}
	if collection, exists := m.CollectionsByName[identifier]; exists {
		return collection, true
	}
	return nil, false
}

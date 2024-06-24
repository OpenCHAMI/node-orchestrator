package smd

import (
	"testing"

	_ "github.com/marcboeker/go-duckdb"

	"github.com/google/uuid"
)

func TestCreateOrUpdateComponents(t *testing.T) {

	// Create the DuckDBSMDStorage instance
	storage, _ := NewDuckDBSMDStorage("")

	// Create a test component
	component := Component{
		UID:                 uuid.New(),
		ID:                  "test-component",
		Type:                "test-type",
		Subtype:             "test-subtype",
		Role:                "test-role",
		SubRole:             "test-sub-role",
		NetType:             "test-net-type",
		Arch:                "test-arch",
		Class:               "test-class",
		State:               "test-state",
		Flag:                "test-flag",
		Enabled:             true,
		SwStatus:            "test-sw-status",
		NID:                 123,
		ReservationDisabled: false,
		Locked:              false,
	}

	// Test creating a new component
	err := storage.CreateOrUpdateComponents([]Component{component})
	if err != nil {
		t.Errorf("failed to create component: %v", err)
	}

	// Test updating an existing component
	component.Enabled = false
	err = storage.CreateOrUpdateComponents([]Component{component})
	if err != nil {
		t.Errorf("failed to update component: %v", err)
	}

	// Test creating multiple components
	components := []Component{
		{
			UID:                 uuid.New(),
			ID:                  "test-component-2",
			Type:                "test-type",
			Subtype:             "test-subtype",
			Role:                "test-role",
			SubRole:             "test-sub-role",
			NetType:             "test-net-type",
			Arch:                "test-arch",
			Class:               "test-class",
			State:               "test-state",
			Flag:                "test-flag",
			Enabled:             true,
			SwStatus:            "test-sw-status",
			NID:                 456,
			ReservationDisabled: false,
			Locked:              false,
		},
		{
			UID:                 uuid.New(),
			ID:                  "test-component-3",
			Type:                "test-type",
			Subtype:             "test-subtype",
			Role:                "test-role",
			SubRole:             "test-sub-role",
			NetType:             "test-net-type",
			Arch:                "test-arch",
			Class:               "test-class",
			State:               "test-state",
			Flag:                "test-flag",
			Enabled:             true,
			SwStatus:            "test-sw-status",
			NID:                 789,
			ReservationDisabled: false,
			Locked:              false,
		},
	}

	err = storage.CreateOrUpdateComponents(components)
	if err != nil {
		t.Errorf("failed to create components: %v", err)
	}
}

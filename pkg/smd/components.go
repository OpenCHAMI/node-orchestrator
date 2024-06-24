package smd

import (
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
)

// Component represents a CSM Component
type Component struct {
	UID                 uuid.UUID        `json:"UID,omitempty" db:"uid"`
	ID                  string           `json:"ID" db:"id" jsonschema:"description=Xname"`
	Type                string           `json:"Type" db:"type"`
	Subtype             string           `json:"Subtype,omitempty" db:"subtype"`
	Role                ComponentRole    `json:"Role,omitempty" db:"role"`
	SubRole             ComponentSubRole `json:"SubRole,omitempty" db:"sub_role"`
	NetType             ComponentNetType `json:"NetType,omitempty" db:"net_type"`
	Arch                ComponentArch    `json:"Arch,omitempty" db:"arch"`
	Class               ComponentClass   `json:"Class,omitempty" db:"class"`
	State               ComponentState   `json:"State,omitempty" db:"state"`
	Flag                ComponentFlag    `json:"Flag,omitempty" db:"flag"`
	Enabled             bool             `json:"Enabled,omitempty" db:"enabled"`
	SwStatus            string           `json:"SoftwareStatus,omitempty" db:"sw_status"`
	NID                 int              `json:"NID,omitempty" db:"nid"`
	ReservationDisabled bool             `json:"ReservationDisabled,omitempty" db:"reservation_disabled"`
	Locked              bool             `json:"Locked,omitempty" db:"locked"`
}

// ComponentState represents the state of an CSM component
type ComponentState string

const (
	StateUnknown   ComponentState = "Unknown"   // The State is unknown.  Appears missing but has not been confirmed as empty.
	StateEmpty     ComponentState = "Empty"     // The location is not populated with a component
	StatePopulated ComponentState = "Populated" // Present (not empty), but no further track can or is being done.
	StateOff       ComponentState = "Off"       // Present but powered off
	StateOn        ComponentState = "On"        // Powered on.  If no heartbeat mechanism is available, its software state may be unknown.

	StateStandby ComponentState = "Standby" // No longer Ready and presumed dead.  It typically means HB has been lost (w/alert).
	StateHalt    ComponentState = "Halt"    // No longer Ready and halted.  OS has been gracefully shutdown or panicked (w/ alert).
	StateReady   ComponentState = "Ready"   // Both On and Ready to provide its expected services, i.e. used for jobs.
)

func (ComponentState) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
		Enum: []interface{}{
			string(StateUnknown),
			string(StateEmpty),
			string(StatePopulated),
			string(StateOff),
			string(StateOn),
			string(StateStandby),
			string(StateHalt),
			string(StateReady),
		},
		Description: "The state of an CSM component",
	}
}

type ComponentFlag string

// Valid flag values.
const (
	FlagUnknown ComponentFlag = "Unknown"
	FlagOK      ComponentFlag = "OK"      // Functioning properly
	FlagWarning ComponentFlag = "Warning" // Continues to operate, but has an issue that may require attention.
	FlagAlert   ComponentFlag = "Alert"   // No longer operating as expected.  The state may also have changed due to error.
	FlagLocked  ComponentFlag = "Locked"  // Another service has reserved this component.
)

func (ComponentFlag) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
		Enum: []interface{}{
			string(FlagUnknown),
			string(FlagOK),
			string(FlagWarning),
			string(FlagAlert),
			string(FlagLocked),
		},
		Description: "The flag of an CSM component",
	}
}

type ComponentRole string

// Valid role values.
const (
	RoleCompute     ComponentRole = "Compute"
	RoleService     ComponentRole = "Service"
	RoleSystem      ComponentRole = "System"
	RoleApplication ComponentRole = "Application"
	RoleStorage     ComponentRole = "Storage"
	RoleManagement  ComponentRole = "Management"
)

func (ComponentRole) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
		Enum: []interface{}{
			string(RoleCompute),
			string(RoleService),
			string(RoleSystem),
			string(RoleApplication),
			string(RoleStorage),
			string(RoleManagement),
		},
		Description: "The role of an CSM component",
	}
}

type ComponentSubRole string

// Valid SubRole values.
const (
	SubRoleMaster  ComponentSubRole = "Master"
	SubRoleWorker  ComponentSubRole = "Worker"
	SubRoleStorage ComponentSubRole = "Storage"
)

func (ComponentSubRole) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
		Enum: []interface{}{
			string(SubRoleMaster),
			string(SubRoleWorker),
			string(SubRoleStorage),
		},
		Description: "The sub-role of an CSM component",
	}
}

type ComponentNetType string

const (
	NetSling      ComponentNetType = "Sling"
	NetInfiniband ComponentNetType = "Infiniband"
	NetEthernet   ComponentNetType = "Ethernet"
	NetOEM        ComponentNetType = "OEM" // Placeholder for non-slingshot
	NetNone       ComponentNetType = "None"
)

func (ComponentNetType) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
		Enum: []interface{}{
			string(NetSling),
			string(NetInfiniband),
			string(NetEthernet),
			string(NetOEM),
			string(NetNone),
		},
		Description: "The network type of an CSM component",
	}
}

type ComponentArch string

const (
	ArchX86     ComponentArch = "X86"
	ArchARM     ComponentArch = "ARM"
	ArchUnknown ComponentArch = "UNKNOWN"
	ArchOther   ComponentArch = "Other"
)

func (ComponentArch) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
		Enum: []interface{}{
			string(ArchX86),
			string(ArchARM),
			string(ArchUnknown),
			string(ArchOther),
		},
		Description: "The architecture of an CSM component",
	}
}

type ComponentClass string

const (
	ClassRiver    ComponentClass = "River"
	ClassMountain ComponentClass = "Mountain"
	ClassHill     ComponentClass = "Hill"
	ClassOther    ComponentClass = "Other"
)

func (ComponentClass) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "string",
		Enum: []interface{}{
			string(ClassRiver),
			string(ClassMountain),
			string(ClassHill),
			string(ClassOther),
		},
		Description: "The class of an CSM component",
	}
}

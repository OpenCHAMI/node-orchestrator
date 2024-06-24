package nodes

import (
	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/xnames"
)

type CloudInitData struct {
	ID         uuid.UUID              `json:"id,omitempty" db:"id"`
	UserData   map[string]interface{} `json:"userdata,omitempty" db:"userdata"`
	MetaData   map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	VendorData map[string]interface{} `json:"vendordata,omitempty" db:"vendordata"`
}

type BootData struct {
	ID                uuid.UUID `json:"id,omitempty" db:"id"`
	KernelURL         string    `json:"kernel_url,omitempty" db:"kernel_url"`
	KernelCommandLine string    `json:"kernel_command_line,omitempty" db:"kernel_command_line"`
	ImageURL          string    `json:"image_url,omitempty" db:"image_url"`
}

type ComputeNode struct {
	ID                uuid.UUID          `json:"id,omitempty" db:"id"`
	Hostname          string             `json:"hostname" binding:"required" db:"hostname"`
	XName             xnames.NodeXname   `json:"xname,omitempty" db:"xname"`
	Architecture      string             `json:"architecture" binding:"required" db:"architecture"`
	BootMac           string             `json:"boot_mac,omitempty" format:"mac-address" db:"boot_mac"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces,omitempty" db:"network_interfaces"`
	BMC               *BMC               `json:"bmc,omitempty" db:"bmc"`
	Description       string             `json:"description,omitempty" db:"description"`
	BootData          *BootData          `json:"boot_data,omitempty" db:"boot_data"`
	CloudInitData     *CloudInitData     `json:"cloud_init_data,omitempty" db:"cloud_init_data"`
	TPMPubKey         string             `json:"tpm_pub_key,omitempty" db:"tpm_pub_key"`
}

type NetworkInterface struct {
	InterfaceName string `json:"interface_name" binding:"required" db:"interface_name"`
	IPv4Address   string `json:"ipv4_address,omitempty" format:"ipv4" db:"ipv4_address"`
	IPv6Address   string `json:"ipv6_address,omitempty" format:"ipv6" db:"ipv6_address"`
	MACAddress    string `json:"mac_address" format:"mac-address" binding:"required" db:"mac_address"`
	Description   string `json:"description,omitempty" db:"description"`
}

type ComputeNodeEvent struct {
	NodeID    uuid.UUID `json:"node_id,omitempty" db:"node_id"`
	EventType string    `json:"event,omitempty" db:"event_type"` // CREATE, UPDATE, DELETE
	EventJSON string    `json:"event_json,omitempty" db:"event_json"`
	Timestamp int64     `json:"timestamp,omitempty" db:"timestamp"`
}

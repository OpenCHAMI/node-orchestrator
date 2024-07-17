package nodes

import (
	"time"

	"github.com/google/uuid"
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
	ID       uuid.UUID `json:"id,omitempty" db:"id"`
	Hostname string    `json:"hostname" binding:"required" db:"hostname"`
	//XName             xnames.NodeXname   `json:"xname,omitempty" db:"xname"`
	Architecture      string             `json:"architecture" binding:"required" db:"architecture"`
	BootMac           string             `json:"boot_mac,omitempty" format:"mac-address" db:"boot_mac"`
	BootIPv4Address   string             `json:"boot_ipv4_address,omitempty" format:"ipv4" db:"boot_ipv4_address"`
	BootIPv6Address   string             `json:"boot_ipv6_address,omitempty" format:"ipv6" db:"boot_ipv6_address"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces,omitempty" db:"network_interfaces"`
	BMC               *BMC               `json:"bmc,omitempty" db:"bmc"`
	Description       string             `json:"description,omitempty" db:"description"`
	BootData          *BootData          `json:"boot_data,omitempty" db:"boot_data"`
	LocationString    string             `json:"location_string,omitempty" db:"location_string"`
	Spec              ComputeNodeSpec    `json:"spec,omitempty" db:"spec"`
	Status            ComputeNodeStatus  `json:"status,omitempty" db:"status"`
}

type ComputeNodeSpec struct {
	Hostname          string             `json:"hostname" binding:"required" db:"hostname"`
	BootMac           string             `json:"boot_mac,omitempty" format:"mac-address" db:"boot_mac"`
	BootIPv4Address   string             `json:"boot_ipv4_address,omitempty" format:"ipv4" db:"boot_ipv4_address"`
	BootIPv6Address   string             `json:"boot_ipv6_address,omitempty" format:"ipv6" db:"boot_ipv6_address"`
	BMCEndpoint       string             `json:"bmc_endpoint,omitempty" db:"bmc_endpoint"`
	BMCUsername       string             `json:"bmc_username,omitempty" db:"bmc_username"`
	BMCPassword       string             `json:"bmc_password,omitempty" db:"bmc_password"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces,omitempty" db:"network_interfaces"`
	BootConfiguration BootData           `json:"boot_configuration,omitempty" db:"boot_configuration"`
}

type ComputeNodeStatus struct {
	PowerState        PowerState             `json:"power_state,omitempty" db:"power_state"`
	BootConfiguration BootConfiguration      `json:"boot_configuration,omitempty" db:"boot_configuration"`
	NetworkInterfaces []NetworkInterface     `json:"network_interfaces,omitempty" db:"network_interfaces"`
	ExtendedData      map[string]interface{} `json:"extended_data,omitempty" db:"extended_data"`
}

type PowerState struct {
	On          bool      `json:"on" db:"on"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
}

type BootConfiguration struct {
	BootData    BootData  `json:"boot_data,omitempty" db:"boot_data"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
}

type NetworkInterface struct {
	InterfaceName   string                 `json:"interface_name" binding:"required" db:"interface_name"`
	IPv4Address     string                 `json:"ipv4_address,omitempty" format:"ipv4" db:"ipv4_address"`
	IPv6Address     string                 `json:"ipv6_address,omitempty" format:"ipv6" db:"ipv6_address"`
	MACAddress      string                 `json:"mac_address" format:"mac-address" binding:"required" db:"mac_address"`
	Description     string                 `json:"description,omitempty" db:"description"`
	Serial          string                 `json:"serial,omitempty" db:"serial_number"`
	Model           string                 `json:"model,omitempty" db:"model"`
	Manufacturer    string                 `json:"manufacturer,omitempty" db:"manufacturer"`
	FirmwareVersion string                 `json:"firmware_version,omitempty" db:"firmware_version"`
	ExtendedData    map[string]interface{} `json:"extended_data,omitempty" db:"extended_data"`
}

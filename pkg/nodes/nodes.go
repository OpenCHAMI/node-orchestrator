package nodes

import (
	"database/sql"

	"github.com/rs/zerolog/log"

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

func CreateNodeTables(db *sql.DB) {
	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("Error connecting to database")
	}
	_, err := db.Exec(createCloudInitDataTableSQL())
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating cloud_init_data table")
	}
	log.Info().Msg("Created cloud_init_data table")

	if _, err := db.Exec(createBootDataTableSQL()); err != nil {
		log.Fatal().Err(err).Msg("Error creating boot_data table")
	}
	log.Info().Msg("Created boot_data table")

	if _, err := db.Exec(createComputeNodeTableSQL()); err != nil {
		log.Fatal().Err(err).Msg("Error creating compute_node table")
	}
	log.Info().Msg("Created compute_node table")

	if _, err := db.Exec(createNetworkInterfaceTableSQL()); err != nil {
		log.Fatal().Err(err).Msg("Error creating network_interface table")
	}
	log.Info().Msg("Created network_interface table")

	if _, err := db.Exec(createComputeNodeEventTableSQL()); err != nil {
		log.Fatal().Err(err).Msg("Error creating compute_node_event table")
	}
	log.Info().Msg("Created compute_node_event table")
}

func createCloudInitDataTableSQL() string {
	return `
CREATE TABLE IF NOT EXISTS cloud_init_data (
	id UUID PRIMARY KEY,
	userdata JSONB,
	metadata JSONB,
	vendordata JSONB
);
`
}

func createBootDataTableSQL() string {
	return `
CREATE TABLE IF NOT EXISTS boot_data (
	id UUID PRIMARY KEY,
	kernel_url TEXT,
	kernel_command_line TEXT,
	image_url TEXT
);
`
}

func createComputeNodeTableSQL() string {
	return `
CREATE TABLE IF NOT EXISTS compute_node (
	id UUID PRIMARY KEY,
	hostname TEXT NOT NULL,
	xname TEXT,
	architecture TEXT NOT NULL,
	boot_mac TEXT,
	description TEXT,
	boot_data_id UUID,
	cloud_init_data_id UUID,
	FOREIGN KEY (boot_data_id) REFERENCES boot_data (id),
	FOREIGN KEY (cloud_init_data_id) REFERENCES cloud_init_data (id)
);
`
}

func createNetworkInterfaceTableSQL() string {
	return `
CREATE TABLE IF NOT EXISTS network_interface (
	interface_name TEXT NOT NULL,
	ipv4_address TEXT,
	ipv6_address TEXT,
	mac_address TEXT NOT NULL,
	description TEXT,
	PRIMARY KEY (mac_address)
);
`

}

func createComputeNodeEventTableSQL() string {
	return `
CREATE TABLE IF NOT EXISTS compute_node_event (
	node_id UUID,
	event_type TEXT,
	event_json TEXT,
	timestamp INTEGER,
	FOREIGN KEY (node_id) REFERENCES compute_node (id)
);
`
}

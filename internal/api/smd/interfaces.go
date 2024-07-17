package smd

// CompEthInterface represents the SMD version of a network interface
type CompEthInterface struct {
	ID         string `json:"ID"`
	Desc       string `json:"Description"`
	MACAddr    string `json:"MACAddress"`
	LastUpdate string `json:"LastUpdate"`
	CompID     string `json:"ComponentID"`
	Type       string `json:"Type"`

	IPAddrs []IPAddressMapping `json:"IPAddresses"`
}

// IPAddressMapping represents an IP Address to network mapping. The network field is optional
type IPAddressMapping struct {
	IPAddr  string `json:"IPAddress"`
	Network string `json:"Network,omitempty"`
}

package nodes

import (
	"github.com/google/uuid"
)

type BMC struct {
	ID          uuid.UUID `json:"id,omitempty" format:"uuid"`
	XName       string    `json:"xname,omitempty"`
	Username    string    `json:"username" jsonschema:"required"`
	Password    string    `json:"password" jsonschema:"required"`
	IPv4Address string    `json:"ipv4_address,omitempty" format:"ipv4"`
	IPv6Address string    `json:"ipv6_address,omitempty" format:"ipv6"`
	MACAddress  string    `json:"mac_address" format:"mac-address" binding:"required"`
	Description string    `json:"description,omitempty"`
}

package main

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/invopop/jsonschema"
)

type NodeXname struct {
	Value string
}

func (n NodeXname) Cabinet() (int, error) {
	if n.Value == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(n.Value).Cabinet, nil
}

func (n NodeXname) Chassis() (int, error) {
	if n.Value == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(n.Value).Chassis, nil
}

func (n NodeXname) Slot() (int, error) {
	if n.Value == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(n.Value).Slot, nil
}

func (n NodeXname) NodePosition() (int, error) {
	if n.Value == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(n.Value).NodePosition, nil
}

func (n NodeXname) BMCPosition() (int, error) {
	if n.Value == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(n.Value).BMCPosition, nil
}

func (n NodeXname) String() string {
	return n.Value
}

type XNameComponents struct {
	Cabinet      int    `json:"cabinet"`
	Chassis      int    `json:"chassis"`
	Slot         int    `json:"slot"`
	BMCPosition  int    `json:"bmc_position"`
	NodePosition int    `json:"node_position"`
	Type         string `json:"type"` // 'n' for node, 'b' for BMC
}

func extractXNameComponents(xname string) XNameComponents {
	var components XNameComponents
	_, err := fmt.Sscanf(xname, "x%dc%ds%db%dn%d", &components.Cabinet, &components.Chassis, &components.Slot, &components.BMCPosition, &components.NodePosition)
	if err == nil {
		components.Type = "n"
		return components
	}
	_, err = fmt.Sscanf(xname, "x%dc%ds%db%d", &components.Cabinet, &components.Chassis, &components.Slot, &components.BMCPosition)
	if err == nil {
		components.Type = "b"
		return components
	}
	return components
}

func (NodeXname) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Title:       "NodeXName",
		Description: "XName for a compute node",
		Pattern:     `^x(\d{3,5})c(\d{1,3})s(\d{1,3})b(\d{1,3})n(\d{1,3})$`,
	}
}

func (xname *NodeXname) UnmarshalJSON(data []byte) error {
	xname.Value = string(data)
	// Remove quotation marks if they exist
	if len(xname.Value) >= 2 && xname.Value[0] == '"' && xname.Value[len(xname.Value)-1] == '"' {
		xname.Value = xname.Value[1 : len(xname.Value)-1]
	}
	return nil
}

func (xname NodeXname) Valid() (bool, error) {
	nodeXnameRegex := regexp.MustCompile(`^x(?P<cabinet>\d{3,5})c(?P<chassis>\d{1,3})s(?P<slot>\d{1,3})b(?P<bmc>\d{1,3})n(?P<node>\d{1,3})$`)
	if !nodeXnameRegex.MatchString(xname.Value) {
		return false, fmt.Errorf("XName does not match regex")
	}

	// Extract the named groups
	match := nodeXnameRegex.FindStringSubmatch(xname.Value)
	result := make(map[string]string)
	for i, name := range nodeXnameRegex.SubexpNames() {
		if i > 0 && i <= len(match) {
			result[name] = match[i]
		}
	}

	// Convert and check chassis number
	chassis, err := strconv.Atoi(result["chassis"])
	if err != nil {
		return false, fmt.Errorf("chassis is not a valid number: %s", result["chassis"])
	}
	if chassis >= 256 {
		return false, fmt.Errorf("chassis number %d exceeds the maximum allowed value of 255", chassis)
	}

	return true, nil
}

func NewNodeXname(xname string) NodeXname {
	return NodeXname{Value: xname}
}

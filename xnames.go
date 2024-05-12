package main

import (
	"fmt"

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
		Pattern:     "^x(?P<cabinet>\\d{3,5})c(?P<chassis>\\d{1,3})s(?P<slot>\\d{1,3})b(?P<bmc>\\d{1,3})n(?P<node>\\d{1,3})$",
	}
}

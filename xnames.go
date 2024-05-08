package main

import "fmt"

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

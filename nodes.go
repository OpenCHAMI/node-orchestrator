package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CloudInitData struct {
	ID uuid.UUID
	// Cloud-init data is a map of maps
	UserData   map[string]interface{} `json:"userdata"`
	MetaData   map[string]interface{} `json:"metadata"`
	VendorData map[string]interface{} `json:"vendordata"`
}

type BootData struct {
	ID                uuid.UUID
	KernelURL         string `json:"kernel_url"`
	KernelCommandLine string `json:"kernel_command_line"`
	ImageURL          string `json:"image_url"`
}

type ComputeNode struct {
	ID                uuid.UUID          `json:"id,omitempty" format:"uuid"`
	Hostname          string             `json:"hostname" binding:"required"`
	XName             string             `json:"xname,omitempty"`
	Architecture      string             `json:"architecture" binding:"required"`
	BootMac           string             `json:"boot_mac,omitempty" format:"mac-address"`
	NetworkInterfaces []NetworkInterface `json:"network_interfaces,omitempty"`
	BMC               *BMC               `json:"bmc,omitempty"`
	Description       string             `json:"description,omitempty"`
	BootData          *BootData          `json:"boot_data,omitempty"`
	CloudInitData     *CloudInitData     `json:"cloud_init_data,omitempty"`
}

type NetworkInterface struct {
	InterfaceName string `json:"interface_name" binding:"required"`
	IPv4Address   string `json:"ipv4_address,omitempty" format:"ipv4"`
	IPv6Address   string `json:"ipv6_address,omitempty" format:"ipv6"`
	MACAddress    string `json:"mac_address" format:"mac-address" binding:"required"`
	Description   string `json:"description,omitempty"`
}

// Cabinet returns the cabinet ID of the node or 0 if the node does not have an XName
func (node *ComputeNode) Cabinet() (int, error) {
	if node.XName == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(node.XName).Cabinet, nil
}

// Chassis returns the Chassis ID of the node or 0 if the node does not have an XName
func (node *ComputeNode) Chassis() (int, error) {
	if node.XName == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(node.XName).Chassis, nil
}

// Slot returns the Slot ID of the node or 0 if the node does not have an XName
func (node *ComputeNode) Slot() (int, error) {
	if node.XName == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(node.XName).Slot, nil
}

// NodePosition returns the Node Position ID of the node or 0 if the node does not have an XName
func (node *ComputeNode) NodePosition() (int, error) {
	if node.XName == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(node.XName).NodePosition, nil
}

// BMCPosition returns the BMC Position ID of the node or 0 if the node does not have an XName
func (node *ComputeNode) BMCPosition() (int, error) {
	if node.XName == "" {
		return 0, fmt.Errorf("node does not have an XName")
	}
	return extractXNameComponents(node.XName).BMCPosition, nil
}

func (a *App) postNode(w http.ResponseWriter, r *http.Request) {
	var newNode ComputeNode

	// Decode the request body into the new node
	if err := json.NewDecoder(r.Body).Decode(&newNode); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// If the XName is supplied, confirm that it is valid and not a duplicate
	if newNode.XName != "" {
		if !isValidNodeXName(newNode.XName) {
			http.Error(w, "invalid XName", http.StatusBadRequest)
		}
		// If the xname isn't empty, check for duplicates which are not allowed
		_, err := a.Storage.LookupComputeNodeByXName(newNode.XName)
		if err == nil {
			http.Error(w, "Compute Node with the same XName already exists", http.StatusBadRequest)
		}
	}

	// If a BMC is supplied, add it to the system
	if newNode.BMC != nil {
		if newNode.BMC.XName != "" && !isValidBMCXName(newNode.BMC.XName) {
			http.Error(w, "invalid BMC XName", http.StatusBadRequest)
		}
		// Check if the BMC alread exists via MAC or XName
		existingBMC, err := a.Storage.LookupBMCByXName(newNode.BMC.XName)
		if err == nil {
			newNode.BMC.ID = existingBMC.ID
		}
		existingBMC, err = a.Storage.LookupBMCByMACAddress(newNode.BMC.MACAddress)
		if err == nil {
			newNode.BMC.ID = existingBMC.ID
		}
		// If the BMC doesn't exist, create a new one
		log.Print("Creating new BMC", newNode.BMC.ID)
		if newNode.BMC.ID == uuid.Nil {
			newNode.BMC.ID = uuid.New()
			a.Storage.SaveBMC(newNode.BMC.ID, *newNode.BMC)
		}
	}

	newNode.ID = uuid.New()
	a.Storage.SaveComputeNode(newNode.ID, newNode)
	json.NewEncoder(w).Encode(newNode)
}

func (a *App) getNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
	if err != nil {
		http.Error(w, "malformed node ID", http.StatusBadRequest)
		return
	}
	node, err := a.Storage.GetComputeNode(nodeID)
	if err != nil {
		http.Error(w, "node not found", http.StatusNotFound)
	} else {
		json.NewEncoder(w).Encode(node)
	}
}

func (a *App) updateNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
	if err != nil {
		http.Error(w, "malformed node ID", http.StatusBadRequest)
		return
	}
	var updateNode ComputeNode
	if err := json.NewDecoder(r.Body).Decode(&updateNode); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if updateNode.XName != "" && !isValidNodeXName(updateNode.XName) {
		http.Error(w, "invalid XName", http.StatusBadRequest)
	}
	err = a.Storage.UpdateComputeNode(nodeID, updateNode)
	if err != nil {
		http.Error(w, "node not found", http.StatusNotFound)
	}
}

func (a *App) deleteNode(w http.ResponseWriter, r *http.Request) {
	nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
	if err != nil {
		http.Error(w, "malformed node ID", http.StatusBadRequest)
		return
	}
	err = a.Storage.DeleteComputeNode(nodeID)
	if err != nil {
		http.Error(w, "node not found", http.StatusNotFound)
	}
}

func isValidNodeXName(xname string) bool {
	// Compile the regular expression. This is the pattern from your requirement.
	re := regexp.MustCompile(`^x(?P<cabinet>\d{3,5})c(?P<chassis>\d{1,3})s(?P<slot>\d{1,3})b(?P<bmc>\d{1,3})n(?P<node>\d{1,3})$`)

	// Use FindStringSubmatch to capture the parts of the xname.
	matches := re.FindStringSubmatch(xname)
	if matches == nil {
		return false
	}

	// Since the cabinet can go up to 100,000 and others up to 255, we need to check these values.
	// The order of subexpressions in matches corresponds to the groups in the regex.
	cabinet, _ := strconv.Atoi(matches[1])
	chassis, _ := strconv.Atoi(matches[2])
	slot, _ := strconv.Atoi(matches[3])
	bmc, _ := strconv.Atoi(matches[4])
	node, _ := strconv.Atoi(matches[5])

	if cabinet > 100000 || chassis >= 256 || slot >= 256 || bmc >= 256 || node >= 256 {
		return false
	}

	return true
}

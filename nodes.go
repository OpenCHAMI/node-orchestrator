package main

import (
	"encoding/json"
	"log"
	"net/http"

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
	XName             NodeXname          `json:"xname,omitempty"`
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

func (a *App) postNode(w http.ResponseWriter, r *http.Request) {
	var newNode ComputeNode

	// Decode the request body into the new node
	if err := json.NewDecoder(r.Body).Decode(&newNode); err != nil {
		log.Print("Error decoding request body", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Print("Decoded new node", newNode)
	// If the XName is supplied, confirm that it is valid and not a duplicate
	if newNode.XName.String() != "" {
		log.Print("Validating XName", newNode.XName.String())
		if _, err := newNode.XName.Valid(); err != nil {
			log.Print("Invalid XName", newNode.XName.String(), err)
			http.Error(w, "Invalid XName"+err.Error(), http.StatusBadRequest)
		}

		// If the xname isn't empty, check for duplicates which are not allowed
		_, err := a.Storage.LookupComputeNodeByXName(newNode.XName.String())
		if err == nil {
			log.Print("Duplicate XName", newNode.XName.String())
			http.Error(w, "Compute Node with the same XName already exists", http.StatusBadRequest)
			return
		}
	}

	// If a BMC is supplied, add it to the system
	if newNode.BMC != nil {
		if newNode.BMC.XName != "" && !isValidBMCXName(newNode.BMC.XName) {
			http.Error(w, "invalid BMC XName", http.StatusBadRequest)
			return
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
	err := a.Storage.SaveComputeNode(newNode.ID, newNode)
	if err != nil {
		log.Print("Error saving node", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Print("New node created", newNode.ID)
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(newNode)
	if err != nil {
		log.Print("Error encoding response", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
	if _, err := updateNode.XName.Valid(); err != nil {
		http.Error(w, "invalid XName "+err.Error(), http.StatusBadRequest)
		return
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

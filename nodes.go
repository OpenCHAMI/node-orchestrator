package main

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
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
	if err := render.DecodeJSON(r.Body, &newNode); err != nil {
		log.WithError(err).Error("Error decoding request body")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// If the XName is supplied, confirm that it is valid and not a duplicate
	if newNode.XName.String() != "" {
		if _, err := newNode.XName.Valid(); err != nil {
			log.Print("Invalid XName ", newNode.XName.String(), err)
			http.Error(w, "Invalid XName "+err.Error(), http.StatusBadRequest)
			return
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
		// Check if the BMC already exists via MAC or XName
		existingBMC, err := a.Storage.LookupBMCByXName(newNode.BMC.XName)
		if err == nil {
			newNode.BMC.ID = existingBMC.ID
		}
		existingBMC, err = a.Storage.LookupBMCByMACAddress(newNode.BMC.MACAddress)
		if err == nil {
			newNode.BMC.ID = existingBMC.ID
		}
		// If the BMC doesn't exist, create a new one
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

	// Log the full details once and only once. This is the "event" of creating a node.
	log.WithFields(log.Fields{
		"node_id":       newNode.ID,
		"node_xname":    newNode.XName.String(),
		"node_hostname": newNode.Hostname,
		"node_arch":     newNode.Architecture,
		"node_boot_mac": newNode.BootMac,
		"bmc_mac":       newNode.BMC.MACAddress,
		"bmc_xname":     newNode.BMC.XName,
		"bmc_id":        newNode.BMC.ID,
		"request_id":    middleware.GetReqID(r.Context()),
	}).Info("Node created")

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, newNode)
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
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, "malformed node ID")
		return
	}

	var updateNode ComputeNode
	if err := render.DecodeJSON(r.Body, &updateNode); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, err.Error())
		return
	}

	if _, err := updateNode.XName.Valid(); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, "invalid XName "+err.Error())
		return
	}

	err = a.Storage.UpdateComputeNode(nodeID, updateNode)
	if err != nil {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, "node not found")
		return
	}
	log.WithFields(log.Fields{
		"node_id":       updateNode.ID,
		"node_xname":    updateNode.XName.String(),
		"node_hostname": updateNode.Hostname,
		"node_arch":     updateNode.Architecture,
		"node_boot_mac": updateNode.BootMac,
		"bmc_mac":       updateNode.BMC.MACAddress,
		"bmc_xname":     updateNode.BMC.XName,
		"bmc_id":        updateNode.BMC.ID,
		"request_id":    middleware.GetReqID(r.Context()),
	}).Info("Node updated")

	render.Status(r, http.StatusOK)
	render.JSON(w, r, updateNode)
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

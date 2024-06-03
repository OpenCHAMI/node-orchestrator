package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/internal/storage"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
	"github.com/rs/zerolog/log"
)

func mustInt(i int, e error) int {
	if e != nil {
		return 0
	}
	return i
}

func postNode(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newNode nodes.ComputeNode

		if err := render.DecodeJSON(r.Body, &newNode); err != nil {
			log.Error().Err(err).Msg("Error decoding request body")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if newNode.XName.String() != "" {
			if _, err := newNode.XName.Valid(); err != nil {
				log.Print("Invalid XName ", newNode.XName.String(), err)
				http.Error(w, "Invalid XName "+err.Error(), http.StatusBadRequest)
				return
			}

			if _, err := storage.LookupComputeNodeByXName(newNode.XName.String()); err == nil {
				log.Print("Duplicate XName", newNode.XName.String())
				http.Error(w, "Compute Node with the same XName already exists", http.StatusBadRequest)
				return
			}
		}

		// Deal with the BMC. If it has been provided already, check if it is valid
		if newNode.BMC != nil {
			if newNode.BMC.XName != "" && !xnames.IsValidBMCXName(newNode.BMC.XName) {
				http.Error(w, "invalid BMC XName", http.StatusBadRequest)
				return
			}

			if existingBMC, err := storage.LookupBMCByXName(newNode.BMC.XName); err == nil {
				newNode.BMC.ID = existingBMC.ID
			} else if existingBMC, err := storage.LookupBMCByMACAddress(newNode.BMC.MACAddress); err == nil {
				newNode.BMC.ID = existingBMC.ID
			}

			if newNode.BMC.ID == uuid.Nil {
				newNode.BMC.ID = uuid.New()
				storage.SaveBMC(newNode.BMC.ID, *newNode.BMC)
			}
		}

		// If the BMC has not been provided, check to see if it can be inferred from the XName and create it if necessary
		if newNode.BMC == nil && newNode.XName.String() != "" {
			bmcXname := fmt.Sprintf("x%dc%ds%db%d",
				mustInt(newNode.XName.Cabinet()),
				mustInt(newNode.XName.Chassis()),
				mustInt(newNode.XName.Slot()),
				mustInt(newNode.XName.BMCPosition()),
			)
			if existingBMC, err := storage.LookupBMCByXName(bmcXname); err == nil {
				newNode.BMC = &existingBMC
			}
			newNode.BMC = &nodes.BMC{
				ID:    uuid.New(),
				XName: bmcXname,
			}
			storage.SaveBMC(newNode.BMC.ID, *newNode.BMC)
		}

		newNode.ID = uuid.New()
		if err := storage.SaveComputeNode(newNode.ID, newNode); err != nil {
			log.Print("Error saving node", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sublog := log.With().
			Str("node_id", newNode.ID.String()).
			Str("xname", newNode.XName.String()).
			Str("hostname", newNode.Hostname).
			Str("arch", newNode.Architecture).
			Str("boot_mac", newNode.BootMac).
			Str("request_id", middleware.GetReqID(r.Context())).
			Logger()

		if newNode.BMC != nil {
			sublog.With().
				Str("bmc_mac", newNode.BMC.MACAddress).
				Str("bmc_xname", newNode.BMC.XName).
				Str("bmc_id", newNode.BMC.ID.String())
		}
		sublog.Info().Msg("Node created")

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, newNode)
	}
}

func getNode(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
		if err != nil {
			log.Error().Err(err).Msg("Error parsing node ID")
			http.Error(w, "malformed node ID", http.StatusBadRequest)
			return
		}
		node, err := storage.GetComputeNode(nodeID)
		if err != nil {
			http.Error(w, "node not found", http.StatusNotFound)
		} else {
			json.NewEncoder(w).Encode(node)
		}
	}
}

func searchNodes(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		xname := query.Get("xname")
		hostname := query.Get("hostname")
		arch := query.Get("arch")
		bootMac := query.Get("boot_mac")
		bmcMac := query.Get("bmc_mac")
		log.Info().
			Str("xname", xname).
			Str("hostname", hostname).
			Str("arch", arch).
			Str("boot_mac", bootMac).
			Str("request_id", middleware.GetReqID(r.Context())).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Msg("Searching nodes")

		nodes, err := storage.SearchComputeNodes(xname, hostname, arch, bootMac, bmcMac)
		if err != nil {
			log.Error().Err(err).Msg("Error searching nodes")
			http.Error(w, "error searching nodes", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(nodes)
	}
}

func updateNode(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
		if err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, "malformed node ID")
			return
		}

		var updateNode nodes.ComputeNode
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

		err = storage.UpdateComputeNode(nodeID, updateNode)
		if err != nil {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, "node not found")
			return
		}

		log.Info().
			Str("node_id", updateNode.ID.String()).
			Str("node_xname", updateNode.XName.String()).
			Str("node_hostname", updateNode.Hostname).
			Str("node_arch", updateNode.Architecture).
			Str("node_boot_mac", updateNode.BootMac).
			Str("bmc_mac", updateNode.BMC.MACAddress).
			Str("bmc_xname", updateNode.BMC.XName).
			Str("bmc_id", updateNode.BMC.ID.String()).
			Str("request_id", middleware.GetReqID(r.Context())).
			Msg("Node updated")

		render.Status(r, http.StatusOK)
		render.JSON(w, r, updateNode)
	}
}

func deleteNode(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nodeID, err := uuid.Parse(chi.URLParam(r, "nodeID"))
		if err != nil {
			http.Error(w, "malformed node ID", http.StatusBadRequest)
			return
		}
		err = storage.DeleteComputeNode(nodeID)
		if err != nil {
			http.Error(w, "node not found", http.StatusNotFound)
		}
	}
}

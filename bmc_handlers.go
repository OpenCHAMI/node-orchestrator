package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/internal/storage"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
)

func postBMC(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newBMC nodes.BMC
		if err := json.NewDecoder(r.Body).Decode(&newBMC); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if newBMC.XName != "" {
			if !xnames.IsValidBMCXName(newBMC.XName) {
				http.Error(w, "invalid XName", http.StatusBadRequest)
			}
			// Check if the XName already exists
			_, err := storage.LookupBMCByXName(newBMC.XName)
			if err == nil {
				http.Error(w, "XName already exists", http.StatusConflict)
				return
			}
		}

		newBMC.ID = uuid.New()
		storage.SaveBMC(newBMC.ID, newBMC)
		json.NewEncoder(w).Encode(newBMC)
	}
}

func updateBMC(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bmcID, err := uuid.Parse(chi.URLParam(r, "bmcID"))
		if err != nil {
			http.Error(w, "malformed node ID", http.StatusBadRequest)
			return
		}
		var updateBMC nodes.BMC
		if err := json.NewDecoder(r.Body).Decode(&updateBMC); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if _, err := storage.GetBMC(bmcID); err == nil {
			updateBMC.ID = bmcID
			storage.SaveBMC(bmcID, updateBMC)
			json.NewEncoder(w).Encode(updateBMC)
		} else {
			http.Error(w, "BMC not found", http.StatusNotFound)
		}

	}
}

func getBMC(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bmcID, err := uuid.Parse(chi.URLParam(r, "bmcID"))
		if err != nil {
			http.Error(w, "malformed node ID", http.StatusBadRequest)
			return
		}
		bmc, err := storage.GetBMC(bmcID)
		if err == nil {
			json.NewEncoder(w).Encode(bmc)
		} else {
			http.Error(w, "node not found", http.StatusNotFound)
		}
	}
}

func deleteBMC(storage storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bmcID, err := uuid.Parse(chi.URLParam(r, "bmcID"))
		if err != nil {
			http.Error(w, "malformed node ID", http.StatusBadRequest)
			return
		}
		err = storage.DeleteBMC(bmcID)
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Deleted BMC with ID: " + bmcID.String()))
		} else {
			http.Error(w, "node not found", http.StatusNotFound)
		}
	}
}

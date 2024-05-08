package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"
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

func (a *App) postBMC(w http.ResponseWriter, r *http.Request) {
	var newBMC BMC
	if err := json.NewDecoder(r.Body).Decode(&newBMC); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if newBMC.XName != "" {
		if !isValidBMCXName(newBMC.XName) {
			http.Error(w, "invalid XName", http.StatusBadRequest)
		}
		// Check if the XName already exists
		_, err := a.Storage.LookupBMCByXName(newBMC.XName)
		if err == nil {
			http.Error(w, "XName already exists", http.StatusConflict)
			return
		}
	}

	newBMC.ID = uuid.New()
	a.Storage.SaveBMC(newBMC.ID, newBMC)
	json.NewEncoder(w).Encode(newBMC)
}

func (a *App) updateBMC(w http.ResponseWriter, r *http.Request) {
	bmcID, err := uuid.Parse(chi.URLParam(r, "bmcID"))
	if err != nil {
		http.Error(w, "malformed node ID", http.StatusBadRequest)
		return
	}
	var updateBMC BMC
	if err := json.NewDecoder(r.Body).Decode(&updateBMC); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if updateBMC.XName != "" && !isValidNodeXName(updateBMC.XName) {
		http.Error(w, "invalid XName", http.StatusBadRequest)
	}
	if _, err := a.Storage.GetBMC(bmcID); err == nil {
		updateBMC.ID = bmcID
		a.Storage.SaveBMC(bmcID, updateBMC)
		json.NewEncoder(w).Encode(updateBMC)
	} else {
		http.Error(w, "BMC not found", http.StatusNotFound)
	}

}

func (a *App) getBMC(w http.ResponseWriter, r *http.Request) {
	bmcID, err := uuid.Parse(chi.URLParam(r, "bmcID"))
	if err != nil {
		http.Error(w, "malformed node ID", http.StatusBadRequest)
		return
	}
	bmc, err := a.Storage.GetBMC(bmcID)
	if err == nil {
		json.NewEncoder(w).Encode(bmc)
	} else {
		http.Error(w, "node not found", http.StatusNotFound)
	}
}

func (a *App) deleteBMC(w http.ResponseWriter, r *http.Request) {
	bmcID, err := uuid.Parse(chi.URLParam(r, "bmcID"))
	if err != nil {
		http.Error(w, "malformed node ID", http.StatusBadRequest)
		return
	}
	err = a.Storage.DeleteBMC(bmcID)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Deleted BMC with ID: " + bmcID.String()))
	} else {
		http.Error(w, "node not found", http.StatusNotFound)
	}
}

func isValidBMCXName(xname string) bool {
	// Compile the regular expression. This is the pattern from your requirement.
	re := regexp.MustCompile(`^x(?P<cabinet>\d{3,5})c(?P<chassis>\d{1,3})s(?P<slot>\d{1,3})b(?P<bmc>\d{1,3})$`)

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

	if cabinet > 100000 || chassis >= 256 || slot >= 256 || bmc >= 256 {
		return false
	}

	return true
}

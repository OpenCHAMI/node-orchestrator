package smd

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
)

type RedfishEndpoint struct {
	UID      uuid.UUID `json:"UID,omitempty" db:"uid"`
	ID       string    `json:"ID" db:"id"`
	Name     string    `json:"Name,omitempty" db:"name"`
	URI      string    `json:"URI,omitempty" db:"uri"`
	Username string    `json:"Username,omitempty" db:"username"`
	Password string    `json:"Password,omitempty" db:"password"`
}

type RedfishEndpointStorage interface {
	GetRedfishEndpoints() ([]RedfishEndpoint, error)
	GetRedfishEndpointByID(id string) (RedfishEndpoint, error)
	CreateOrUpdateRedfishEndpoints(endpoints []RedfishEndpoint) error
	DeleteRedfishEndpointByID(id string) error
}

// Handler to retrieve all Redfish endpoints
func getRedfishEndpoints(storage RedfishEndpointStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		endpoints, err := storage.GetRedfishEndpoints()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(endpoints)
	}
}

// Handler to retrieve a specific Redfish endpoint by its ID
func getRedfishEndpointByID(storage RedfishEndpointStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		endpoint, err := storage.GetRedfishEndpointByID(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(endpoint)
	}
}

// Handler to create or update Redfish endpoints
func createOrUpdateRedfishEndpoints(storage RedfishEndpointStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var endpoints []RedfishEndpoint
		if err := json.NewDecoder(r.Body).Decode(&endpoints); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for i := range endpoints {
			if endpoints[i].UID == uuid.Nil {
				endpoints[i].UID = uuid.New()
			}
		}

		if err := storage.CreateOrUpdateRedfishEndpoints(endpoints); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// Handler to delete a specific Redfish endpoint by its ID
func deleteRedfishEndpointByID(storage RedfishEndpointStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := storage.DeleteRedfishEndpointByID(id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func NewRedfishRouter(storage RedfishEndpointStorage) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/Inventory/RedfishEndpoints", func(r chi.Router) {
		r.Get("/", getRedfishEndpoints(storage))
		r.Post("/", createOrUpdateRedfishEndpoints(storage))

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", getRedfishEndpointByID(storage))
			r.Delete("/", deleteRedfishEndpointByID(storage))
		})
	})

	return r
}

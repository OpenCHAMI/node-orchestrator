package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	smd "github.com/openchami/node-orchestrator/pkg/smd"
	"github.com/xeipuuv/gojsonschema"
)

// Initialize the schema globally
var componentSchemaLoader gojsonschema.JSONLoader

type SMDStorage interface {
	GetComponents() ([]smd.Component, error)
	GetComponentByXname(xname string) (smd.Component, error)
	GetComponentByNID(nid int) (smd.Component, error)
	GetComponentByUID(uid uuid.UUID) (smd.Component, error)
	QueryComponents(xname string, params map[string]string) ([]smd.Component, error)
	CreateOrUpdateComponents(components []smd.Component) error
	DeleteComponents() error
	DeleteComponentByXname(xname string) error
	UpdateComponentData(xnames []string, data map[string]interface{}) error
}

// ValidationErrorResponse represents a detailed error response
type ValidationErrorResponse struct {
	Message string `json:"message"`
}

func validateWithSchema(documentLoader gojsonschema.JSONLoader) []*ValidationErrorResponse {
	result, err := gojsonschema.Validate(componentSchemaLoader, documentLoader)
	if err != nil {
		return []*ValidationErrorResponse{{Message: err.Error()}}
	}

	var errors []*ValidationErrorResponse
	if !result.Valid() {
		for _, desc := range result.Errors() {
			errors = append(errors, &ValidationErrorResponse{Message: desc.Description()})
		}
	}
	return errors
}

func getComponents(storage SMDStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		components, err := storage.GetComponents()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(components)
	}
}

func getComponentByXname(storage SMDStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		xname := chi.URLParam(r, "xname")
		component, err := storage.GetComponentByXname(xname)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(component)
	}
}

func createUpdateComponents(storage SMDStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var components []smd.Component
		if err := json.NewDecoder(r.Body).Decode(&components); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate each component
		for _, component := range components {
			documentLoader := gojsonschema.NewGoLoader(component)
			if errs := validateWithSchema(documentLoader); len(errs) > 0 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs)
				return
			}
		}

		if err := storage.CreateOrUpdateComponents(components); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deleteComponents(storage SMDStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := storage.DeleteComponents(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func deleteComponentByXname(storage SMDStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		xname := chi.URLParam(r, "xname")
		if err := storage.DeleteComponentByXname(xname); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func updateComponentData(storage SMDStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Xnames []string               `json:"xnames"`
			Data   map[string]interface{} `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate the request
		documentLoader := gojsonschema.NewGoLoader(request)
		if errs := validateWithSchema(documentLoader); len(errs) > 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errs)
			return
		}

		if err := storage.UpdateComponentData(request.Xnames, request.Data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func NewRouter(storage SMDStorage) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Route("/State/Components", func(r chi.Router) {
		r.Get("/", getComponents(storage))
		r.Post("/", createUpdateComponents(storage))
		r.Delete("/", deleteComponents(storage))

		r.Route("/{xname}", func(r chi.Router) {
			r.Get("/", getComponentByXname(storage))
			r.Put("/", createUpdateComponents(storage))
			r.Delete("/", deleteComponentByXname(storage))
		})

		r.Route("/BulkStateData", func(r chi.Router) {
			r.Patch("/", updateComponentData(storage))
		})

		r.Route("/BulkFlagOnly", func(r chi.Router) {
			r.Patch("/", updateComponentData(storage))
		})

		r.Route("/BulkEnabled", func(r chi.Router) {
			r.Patch("/", updateComponentData(storage))
		})

		r.Route("/BulkSoftwareStatus", func(r chi.Router) {
			r.Patch("/", updateComponentData(storage))
		})

		r.Route("/BulkRole", func(r chi.Router) {
			r.Patch("/", updateComponentData(storage))
		})

		r.Route("/BulkNID", func(r chi.Router) {
			r.Patch("/", updateComponentData(storage))
		})

		r.Route("/ByNID/{nid}", func(r chi.Router) {
			r.Get("/", getComponentByXname(storage))
		})

		r.Route("/Query/{xname}", func(r chi.Router) {
			r.Get("/", getComponentByXname(storage))
		})

		r.Route("/Query", func(r chi.Router) {
			r.Post("/", createUpdateComponents(storage))
		})

		r.Route("/ByNID/Query", func(r chi.Router) {
			r.Post("/", createUpdateComponents(storage))
		})
	})

	return r
}

func SMDComponentRoutes(storage SMDStorage) chi.Router {
	// Generate JSON schema for smd.Component struct
	reflector := jsonschema.Reflector{}
	componentSchema := reflector.Reflect(&smd.Component{})

	// Convert schema to JSON
	schemaJSON, err := json.Marshal(componentSchema)
	if err != nil {
		panic(err)
	}

	// Initialize the JSON schema loader
	componentSchemaLoader = gojsonschema.NewBytesLoader(schemaJSON)

	r := NewRouter(storage)
	return r
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
	"github.com/rs/zerolog/log"
)

func createCollection(manager *nodes.CollectionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var collection nodes.NodeCollection
		if err := json.NewDecoder(r.Body).Decode(&collection); err != nil {
			log.Error().
				Err(fmt.Errorf("error binding collection: %v", err)).
				Msg("Error binding collection")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		claims, err := extract_claims(r)
		if err != nil {
			log.Error().
				Err(fmt.Errorf("error extracting claims: %w", err)).
				Msg("Error extracting claims")
		}

		collection.Owner = uuid.MustParse(claims["sub"].(string))
		collection.CreatorSubject = claims["sub"].(string)

		if err := manager.CreateCollection(&collection); err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		log.Info().
			Str("collection_id", collection.ID.String()).
			Str("owner", collection.Owner.String()).
			Str("creator", collection.CreatorSubject).
			Str("description", collection.Description).
			Str("name", collection.Name).
			Str("type", collection.Type.String()).
			Strs("nodes", xnames.XnameSliceString(collection.Nodes)).
			Str("alias", collection.Alias).
			Str("request_id", middleware.GetReqID(r.Context())).
			Str("request_uri", r.RequestURI).
			Str("jwt_subject", claims["sub"].(string)).
			Msg("Collection created")

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, collection)
	}
}

func extract_claims(r *http.Request) (map[string]interface{}, error) {
	_, claims, err := jwtauth.FromContext(r.Context())
	if err != nil {
		return map[string]interface{}{}, err
	}
	return claims, nil
}

func getCollection(manager *nodes.CollectionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier := chi.URLParam(r, "identifier")
		collection, exists := manager.GetCollection(identifier)
		if !exists {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}
		render.JSON(w, r, collection)
	}
}

func updateCollection(manager *nodes.CollectionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier := chi.URLParam(r, "identifier")
		claims, err := extract_claims(r)
		if err != nil {
			log.Error().Err(err).Msg("Error extracting claims")
		}
		var collection nodes.NodeCollection
		if err := render.Bind(r, &collection); err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		existingCollection, exists := manager.GetCollection(identifier)
		if !exists {
			http.Error(w, "Collection not found", http.StatusNotFound)
			return
		}

		collection.ID = existingCollection.ID // Ensure the ID remains the same

		if err := manager.UpdateCollection(&collection); err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}
		log.Info().
			Str("collection_id", collection.ID.String()).
			Str("owner", collection.Owner.String()).
			Str("creator", collection.CreatorSubject).
			Str("description", collection.Description).
			Str("name", collection.Name).
			Str("type", collection.Type.String()).
			Strs("nodes", xnames.XnameSliceString(collection.Nodes)).
			Str("alias", collection.Alias).
			Str("request_id", middleware.GetReqID(r.Context())).
			Str("request_uri", r.RequestURI).
			Str("jwt_subject", claims["sub"].(string)).
			Msg("Collection updated")

		render.Status(r, http.StatusOK)
		render.JSON(w, r, collection)
	}
}

func deleteCollection(manager *nodes.CollectionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier := chi.URLParam(r, "identifier")
		identifierUUID, err := uuid.Parse(identifier)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing identifier")
			render.Render(w, r, ErrInvalidRequest(err))
			return
		}

		if err := manager.DeleteCollection(identifierUUID); err != nil {
			log.Error().Err(err).Msg("Error deleting collection")
			render.Render(w, r, ErrInternalServer)
			return
		}

		render.Status(r, http.StatusNoContent)
	}
}

// ErrResponse renderer type for handling all sorts of errors.
//
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.
type ErrResponse struct {
	Err            error `json:"-"` // low-level runtime error
	HTTPStatusCode int   `json:"-"` // http response status code

	StatusText string `json:"status"`          // user-level status message
	AppCode    int64  `json:"code,omitempty"`  // application-specific error code
	ErrorText  string `json:"error,omitempty"` // application-level error message, for debugging
}

func (e *ErrResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func ErrInvalidRequest(err error) render.Renderer {
	return &ErrResponse{
		Err:            err,
		HTTPStatusCode: 400,
		StatusText:     "Invalid request.",
		ErrorText:      err.Error(),
	}
}

var ErrNotFound = &ErrResponse{HTTPStatusCode: 404, StatusText: "Resource not found."}
var ErrInternalServer = &ErrResponse{HTTPStatusCode: 500, StatusText: "Internal server error."}

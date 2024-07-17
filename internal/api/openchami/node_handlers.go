package openchami

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/openchami/node-orchestrator/internal/storage"
	openchami_middleware "github.com/openchami/node-orchestrator/pkg/middleware"
	"github.com/openchami/node-orchestrator/pkg/nodes"
	"github.com/openchami/node-orchestrator/pkg/xnames"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func mustInt(i int, e error) int {
	if e != nil {
		return 0
	}
	return i
}

func postNode(storage storage.NodeStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newNode nodes.ComputeNode
		var nodeXName xnames.NodeXname

		if err := render.DecodeJSON(r.Body, &newNode); err != nil {
			log.Error().Err(err).Msg("Error decoding request body")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// If the LocationString validates as an Xname, check if it is valid
		if newNode.LocationString != "" {
			nodeXName = xnames.NodeXname{Value: newNode.LocationString}
			if _, err := nodeXName.Valid(); err != nil {
				log.Print("Invalid XName ", nodeXName.String(), err)
				http.Error(w, "Invalid XName "+err.Error(), http.StatusBadRequest)
				return
			}

			if _, err := storage.LookupComputeNodeByXName(nodeXName.String()); err == nil {
				log.Print("Duplicate XName", nodeXName.String())
				http.Error(w, "Compute Node with the same XName already exists", http.StatusBadRequest)
				return
			}
		}

		// Deal with the BMC. If it has been provided already, check if it is valid
		if newNode.BMC != nil {
			if newNode.BMC.LocationString != "" && !xnames.IsValidBMCXName(newNode.BMC.LocationString) {
				http.Error(w, "invalid BMC XName", http.StatusBadRequest)
				return
			}

			if existingBMC, err := storage.LookupBMCByXName(newNode.BMC.LocationString); err == nil {
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
		if newNode.BMC == nil && nodeXName.String() != "" {
			bmcXname := fmt.Sprintf("x%dc%ds%db%d",
				mustInt(nodeXName.Cabinet()),
				mustInt(nodeXName.Chassis()),
				mustInt(nodeXName.Slot()),
				mustInt(nodeXName.BMCPosition()),
			)
			if existingBMC, err := storage.LookupBMCByXName(bmcXname); err == nil {
				newNode.BMC = &existingBMC
			}
			newNode.BMC = &nodes.BMC{
				ID:             uuid.New(),
				LocationString: xnames.BMCXname{Value: bmcXname}.String(),
			}
			storage.SaveBMC(newNode.BMC.ID, *newNode.BMC)
		}

		newNode.ID = uuid.New()
		if err := storage.SaveComputeNode(newNode.ID, newNode); err != nil {
			log.Print("Error saving node", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sublogger := r.Context().Value(openchami_middleware.LoggerKey).(*zerolog.Logger)

		sublog := sublogger.With().
			Str("node_id", newNode.ID.String()).
			Str("xname", nodeXName.String()).
			Str("hostname", newNode.Hostname).
			Str("arch", newNode.Architecture).
			Str("boot_mac", newNode.BootMac).
			Str("event_type", "create_node").
			Logger()

		if newNode.BMC != nil {
			sublog.With().
				Str("bmc_mac", newNode.BMC.MACAddress).
				Str("bmc_xname", newNode.BMC.LocationString).
				Str("bmc_id", newNode.BMC.ID.String()).
				Logger()
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, newNode)
	}
}

func getNode(storage storage.NodeStorage) http.HandlerFunc {
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

func searchNodes(myStorage storage.NodeStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		query := r.URL.Query()
		var searchOptions []storage.NodeSearchOption
		xname := query.Get("xname")
		if xname != "" {
			searchOptions = append(searchOptions, storage.WithXName(xname))
		}
		hostname := query.Get("hostname")
		if hostname != "" {
			searchOptions = append(searchOptions, storage.WithHostname(hostname))
		}
		arch := query.Get("arch")
		if arch != "" {
			searchOptions = append(searchOptions, storage.WithArch(arch))
		}
		bootMac := query.Get("boot_mac")
		if bootMac != "" {
			searchOptions = append(searchOptions, storage.WithBootMAC(bootMac))
		}
		bmcMac := query.Get("bmc_mac")
		if bmcMac != "" {
			searchOptions = append(searchOptions, storage.WithBMCMAC(bmcMac))
		}
		missingIPV4 := query.Get("missingIPV4")
		if missingIPV4 == "true" {
			searchOptions = append(searchOptions, storage.WithMissingIPV4())
		}
		missingIPV6 := query.Get("missingIPV4")
		if missingIPV6 == "true" {
			searchOptions = append(searchOptions, storage.WithMissingIPV6())
		}
		log.Debug().
			Str("xname", xname).
			Str("hostname", hostname).
			Str("arch", arch).
			Str("boot_mac", bootMac).
			Str("request_id", middleware.GetReqID(r.Context())).
			Str("path", r.URL.Path).
			Str("query", r.URL.RawQuery).
			Msg("Dispatching ComputeNode search to Storage")

		nodes, err := myStorage.SearchComputeNodes(searchOptions...)
		if err != nil {
			log.Error().Err(err).Msg("Error searching nodes")
			http.Error(w, "error searching nodes", http.StatusInternalServerError)
			return
		}

		// If the logging middleware is loaded, add event details
		requestLogger, ok := r.Context().Value(openchami_middleware.LoggerKey).(*zerolog.Logger)
		if ok {
			*requestLogger = requestLogger.With().
				Int("num_nodes", len(nodes)).
				Str("event_type", "search_nodes").
				Logger()

		}

		json.NewEncoder(w).Encode(nodes)
	}
}

func updateNode(storage storage.NodeStorage) http.HandlerFunc {
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

		// If the LocationString validates as an Xname, check if it is valid
		if !(updateNode.LocationString == xnames.NodeXname{Value: updateNode.LocationString}.String()) {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, "invalid XName "+updateNode.LocationString)
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
			Str("node_xname", updateNode.LocationString).
			Str("node_hostname", updateNode.Hostname).
			Str("node_arch", updateNode.Architecture).
			Str("node_boot_mac", updateNode.BootMac).
			Str("bmc_mac", updateNode.BMC.MACAddress).
			Str("bmc_xname", updateNode.BMC.LocationString).
			Str("bmc_id", updateNode.BMC.ID.String()).
			Str("request_id", middleware.GetReqID(r.Context())).
			Msg("Node updated")

		render.Status(r, http.StatusOK)
		render.JSON(w, r, updateNode)
	}
}

func deleteNode(storage storage.NodeStorage) http.HandlerFunc {
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

func NodeRoutes(myStorage storage.NodeStorage, authMiddlewares []func(http.Handler) http.Handler) chi.Router {
	// Create a new collection manager for node collections
	manager := nodes.NewCollectionManager()
	// Add a mutual exclusivity constraint to the manager that prevents a node from being in multipe partitions or multiple tenants.  Use the xname as the key.
	manager.AddConstraint(nodes.DefaultType, &nodes.MutualExclusivityConstraint{ExistingNodes: make(map[xnames.NodeXname]uuid.UUID)})
	manager.AddConstraint(nodes.PartitionType, &nodes.MutualExclusivityConstraint{ExistingNodes: make(map[xnames.NodeXname]uuid.UUID)})
	manager.AddConstraint(nodes.TenantType, &nodes.MutualExclusivityConstraint{ExistingNodes: make(map[xnames.NodeXname]uuid.UUID)})

	// Create a router for both protected and unprotected routes
	r := chi.NewRouter()

	// ComputeNode routes
	r.With(authMiddlewares...).Put("/ComputeNode/{nodeID}", updateNode(myStorage))
	r.With(authMiddlewares...).Post("/ComputeNode/{nodeID}", updateNode(myStorage))
	r.With(authMiddlewares...).Post("/ComputeNode", postNode(myStorage))
	r.With(authMiddlewares...).Delete("/ComputeNode/{nodeID}", deleteNode(myStorage))

	// BMC routes
	r.With(authMiddlewares...).Post("/bmc", postBMC(myStorage))
	r.With(authMiddlewares...).Put("/bmc/{bmcID}", updateBMC(myStorage))
	r.With(authMiddlewares...).Delete("/bmc/{bmcID}", deleteBMC(myStorage))

	// NodeCollection routes
	r.With(authMiddlewares...).Post("/NodeCollection", createCollection(manager))
	r.With(authMiddlewares...).Put("/NodeCollection/{identifier}", updateCollection(manager))
	r.With(authMiddlewares...).Delete("/NodeCollection/{identifier}", deleteCollection(manager))

	// Unprotected routes
	r.Get("/ComputeNode/{nodeID}", getNode(myStorage))
	r.Get("/ComputeNode", searchNodes(myStorage))
	r.Get("/bmc/{bmcID}", getBMC(myStorage))
	r.Get("/NodeCollection/{identifier}", getCollection(manager))

	return r
}

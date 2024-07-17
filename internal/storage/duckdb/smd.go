package duckdb

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/openchami/node-orchestrator/internal/api/smd"
)

func initComponentTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS components (
		uid UUID,
		id TEXT PRIMARY KEY,
		type TEXT,
		subtype TEXT,
		role TEXT,
		sub_role TEXT,
		net_type TEXT,
		arch TEXT,
		class TEXT,
		state TEXT,
		flag TEXT,
		enabled BOOLEAN,
		sw_status TEXT,
		nid INTEGER,
		reservation_disabled BOOLEAN,
		locked BOOLEAN
	)`,
		`CREATE TABLE IF NOT EXISTS redfish_endpoints (
		id TEXT PRIMARY KEY,
		name TEXT,
		uri TEXT,
		username TEXT,
		password TEXT
	)`}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (s *DuckDBStorage) GetComponents() ([]smd.Component, error) {
	query := "SELECT * FROM components"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []smd.Component
	for rows.Next() {
		var c smd.Component
		if err := rows.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
			return nil, err
		}
		components = append(components, c)
	}
	return components, nil
}

func (s *DuckDBStorage) GetComponentByXname(xname string) (smd.Component, error) {
	query := "SELECT * FROM components WHERE id = ?"
	row := s.db.QueryRow(query, xname)

	var c smd.Component
	if err := row.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
		return c, err
	}
	return c, nil
}

func (s *DuckDBStorage) GetComponentByNID(nid int) (smd.Component, error) {
	query := "SELECT * FROM components WHERE nid = ?"
	row := s.db.QueryRow(query, nid)

	var c smd.Component
	if err := row.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
		return c, err
	}
	return c, nil
}

func (s *DuckDBStorage) GetComponentByUID(uid uuid.UUID) (smd.Component, error) {
	query := "SELECT * FROM components WHERE uid = ?"
	row := s.db.QueryRow(query, uid)

	var c smd.Component
	if err := row.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
		if err == sql.ErrNoRows {
			return c, fmt.Errorf("component not found")
		}
		return c, err
	}
	return c, nil
}

func (s *DuckDBStorage) QueryComponents(xname string, params map[string]string) ([]smd.Component, error) {
	query := "SELECT * FROM components WHERE id = ?"
	args := []interface{}{xname}

	for k, v := range params {
		query += fmt.Sprintf(" AND %s = ?", k)
		args = append(args, v)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []smd.Component
	for rows.Next() {
		var c smd.Component
		if err := rows.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
			return nil, err
		}
		components = append(components, c)
	}
	return components, nil
}

func (s *DuckDBStorage) CreateOrUpdateComponents(components []smd.Component) error {
	for _, c := range components {

		var existingComponent smd.Component
		var err error

		// Check if component already exists by xname
		if c.ID != "" {
			existingComponent, err = s.GetComponentByXname(c.ID)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			// Check if it exists by uuid
		} else if c.UID != uuid.Nil {
			existingComponent, err = s.GetComponentByUID(c.UID)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			// if it doesn't exist, create
		} else {
			existingComponent = smd.Component{}
		}

		// If component exists, update it
		if existingComponent.UID != uuid.Nil {
			query := `
			UPDATE components SET
			uid = ?,
			type = ?,
			subtype = ?,
			role = ?,
			sub_role = ?,
			net_type = ?,
			arch = ?,
			class = ?,
			state = ?,
			flag = ?,
			enabled = ?,
			sw_status = ?,
			nid = ?,
			reservation_disabled = ?,
			locked = ?
			WHERE id = ?`

			_, err := s.db.Exec(query, c.UID, c.Type, c.Subtype, c.Role, c.SubRole, c.NetType, c.Arch, c.Class, c.State, c.Flag, c.Enabled, c.SwStatus, c.NID, c.ReservationDisabled, c.Locked, c.ID)
			if err != nil {
				return err
			}
		} else {
			// If component does not exist, create it
			c.UID = uuid.New()
			query := `
			INSERT INTO components (uid, id, type, subtype, role, sub_role, net_type, arch, class, state, flag, enabled, sw_status, nid, reservation_disabled, locked)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

			_, err := s.db.Exec(query, c.UID, c.ID, c.Type, c.Subtype, c.Role, c.SubRole, c.NetType, c.Arch, c.Class, c.State, c.Flag, c.Enabled, c.SwStatus, c.NID, c.ReservationDisabled, c.Locked)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *DuckDBStorage) DeleteComponents() error {
	query := "DELETE FROM components"
	_, err := s.db.Exec(query)
	return err
}

func (s *DuckDBStorage) DeleteComponentByXname(xname string) error {
	query := "DELETE FROM components WHERE id = ?"
	_, err := s.db.Exec(query, xname)
	return err
}

func (s *DuckDBStorage) UpdateComponentData(xnames []string, data map[string]interface{}) error {
	setClauses := []string{}
	args := []interface{}{}

	for k, v := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}
	args = append(args, strings.Join(xnames, ","))

	query := fmt.Sprintf("UPDATE components SET %s WHERE id IN (?)", strings.Join(setClauses, ", "))
	_, err := s.db.Exec(query, args...)
	return err
}

func (s *DuckDBStorage) GetRedfishEndpoints() ([]smd.RedfishEndpoint, error) {
	query := "SELECT * FROM redfish_endpoints"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var endpoints []smd.RedfishEndpoint
	for rows.Next() {
		var e smd.RedfishEndpoint
		if err := rows.Scan(&e.ID, &e.Name, &e.URI, &e.User, &e.Password); err != nil {
			return nil, err
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, nil
}

func (s *DuckDBStorage) GetRedfishEndpointByID(id string) (smd.RedfishEndpoint, error) {
	query := "SELECT * FROM redfish_endpoints WHERE id = ?"
	row := s.db.QueryRow(query, id)
	var e smd.RedfishEndpoint
	if err := row.Scan(&e.ID, &e.Name, &e.URI, &e.User, &e.Password); err != nil {
		return e, err
	}
	return e, nil
}

func (s *DuckDBStorage) CreateOrUpdateRedfishEndpoints(endpoints []smd.RedfishEndpoint) error {
	for _, e := range endpoints {
		var existingEndpoint smd.RedfishEndpoint
		var err error
		// Check if endpoint already exists by ID
		if e.ID != "" {
			existingEndpoint, err = s.GetRedfishEndpointByID(e.ID)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
		} else {
			existingEndpoint = smd.RedfishEndpoint{}
		}
		// If endpoint exists, update it
		if existingEndpoint.ID != "" {
			query := `
			UPDATE redfish_endpoints SET
			name = ?,
			url = ?,
			username = ?,
			password = ?
			WHERE id = ?`
			_, err := s.db.Exec(query, e.Name, e.URI, e.User, e.Password, e.ID)
			if err != nil {
				return err
			}
		} else {
			// If endpoint does not exist, create it
			query := `
			INSERT INTO redfish_endpoints (id, name, url, username, password)
			VALUES (?, ?, ?, ?, ?)`
			_, err := s.db.Exec(query, e.ID, e.Name, e.URI, e.User, e.Password)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *DuckDBStorage) DeleteRedfishEndpointByID(id string) error {
	query := "DELETE FROM redfish_endpoints WHERE id = ?"
	_, err := s.db.Exec(query, id)
	return err
}

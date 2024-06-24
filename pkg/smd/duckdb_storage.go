package smd

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
)

type DuckDBSMDStorage struct {
	db *sql.DB
}

func NewDuckDBSMDStorage(dataSourceName string) (*DuckDBSMDStorage, error) {
	db, err := sql.Open("duckdb", dataSourceName)
	if err != nil {
		return nil, err
	}

	storage := &DuckDBSMDStorage{db: db}
	if err := storage.initDB(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *DuckDBSMDStorage) initDB() error {
	query := `
	CREATE TABLE IF NOT EXISTS components (
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
	)`
	_, err := s.db.Exec(query)
	return err
}

func (s *DuckDBSMDStorage) GetComponents() ([]Component, error) {
	query := "SELECT * FROM components"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []Component
	for rows.Next() {
		var c Component
		if err := rows.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
			return nil, err
		}
		components = append(components, c)
	}
	return components, nil
}

func (s *DuckDBSMDStorage) GetComponentByXname(xname string) (Component, error) {
	query := "SELECT * FROM components WHERE id = ?"
	row := s.db.QueryRow(query, xname)

	var c Component
	if err := row.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
		return c, err
	}
	return c, nil
}

func (s *DuckDBSMDStorage) GetComponentByNID(nid int) (Component, error) {
	query := "SELECT * FROM components WHERE nid = ?"
	row := s.db.QueryRow(query, nid)

	var c Component
	if err := row.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
		return c, err
	}
	return c, nil
}

func (s *DuckDBSMDStorage) GetComponentByUID(uid uuid.UUID) (Component, error) {
	query := "SELECT * FROM components WHERE uid = ?"
	row := s.db.QueryRow(query, uid)

	var c Component
	if err := row.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
		if err == sql.ErrNoRows {
			return c, fmt.Errorf("component not found")
		}
		return c, err
	}
	return c, nil
}

func (s *DuckDBSMDStorage) QueryComponents(xname string, params map[string]string) ([]Component, error) {
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

	var components []Component
	for rows.Next() {
		var c Component
		if err := rows.Scan(&c.UID, &c.ID, &c.Type, &c.Subtype, &c.Role, &c.SubRole, &c.NetType, &c.Arch, &c.Class, &c.State, &c.Flag, &c.Enabled, &c.SwStatus, &c.NID, &c.ReservationDisabled, &c.Locked); err != nil {
			return nil, err
		}
		components = append(components, c)
	}
	return components, nil
}

func (s *DuckDBSMDStorage) CreateOrUpdateComponents(components []Component) error {
	for _, c := range components {

		var existingComponent Component
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
			existingComponent = Component{}
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

func (s *DuckDBSMDStorage) DeleteComponents() error {
	query := "DELETE FROM components"
	_, err := s.db.Exec(query)
	return err
}

func (s *DuckDBSMDStorage) DeleteComponentByXname(xname string) error {
	query := "DELETE FROM components WHERE id = ?"
	_, err := s.db.Exec(query, xname)
	return err
}

func (s *DuckDBSMDStorage) UpdateComponentData(xnames []string, data map[string]interface{}) error {
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

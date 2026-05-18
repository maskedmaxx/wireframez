package schema

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// FieldDef describes a single field in a schema
type FieldDef struct {
	Name string `json:"name"`
	Type string `json:"type"` // "int32", "int64", "float32", "float64", "bool", "string"
}

// Schema is a versioned definition of a payload shape
type Schema struct {
	ID        int        `json:"id"`
	Name      string     `json:"name"`
	Version   int        `json:"version"`
	Fields    []FieldDef `json:"fields"`
	CreatedAt time.Time  `json:"created_at"`
}

// Store handles all schema persistence
type Store struct {
	db *sql.DB
}

// NewStore opens a connection to Postgres and returns a Store
func NewStore(connStr string) (*Store, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Store{db: db}, nil
}

// Register saves a new schema version. If the name already exists,
// it auto-increments the version.
func (s *Store) Register(name string, fields []FieldDef) (*Schema, error) {
	fieldsJSON, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("marshal fields: %w", err)
	}

	var nextVersion int
	err = s.db.QueryRow(`
		SELECT COALESCE(MAX(version), 0) + 1 FROM schemas WHERE name = $1
	`, name).Scan(&nextVersion)
	if err != nil {
		return nil, fmt.Errorf("get next version: %w", err)
	}

	row := s.db.QueryRow(`
		INSERT INTO schemas (name, version, fields)
		VALUES ($1, $2, $3)
		RETURNING id, name, version, fields, created_at
	`, name, nextVersion, fieldsJSON)

	return scanSchema(row)
}

// GetLatest returns the most recent version of a schema by name
func (s *Store) GetLatest(name string) (*Schema, error) {
	row := s.db.QueryRow(`
		SELECT id, name, version, fields, created_at
		FROM schemas
		WHERE name = $1
		ORDER BY version DESC
		LIMIT 1
	`, name)

	schema, err := scanSchema(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("schema %q not found", name)
	}
	return schema, err
}

// GetVersion returns a specific version of a schema
func (s *Store) GetVersion(name string, version int) (*Schema, error) {
	row := s.db.QueryRow(`
		SELECT id, name, version, fields, created_at
		FROM schemas
		WHERE name = $1 AND version = $2
	`, name, version)

	schema, err := scanSchema(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("schema %q version %d not found", name, version)
	}
	return schema, err
}

// List returns all schemas (latest version of each)
func (s *Store) List() ([]Schema, error) {
	rows, err := s.db.Query(`
		SELECT DISTINCT ON (name) id, name, version, fields, created_at
		FROM schemas
		ORDER BY name, version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}
	defer rows.Close()

	var schemas []Schema
	for rows.Next() {
		var sc Schema
		var fieldsJSON []byte
		err := rows.Scan(&sc.ID, &sc.Name, &sc.Version, &fieldsJSON, &sc.CreatedAt)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(fieldsJSON, &sc.Fields); err != nil {
			return nil, err
		}
		schemas = append(schemas, sc)
	}
	return schemas, rows.Err()
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

func scanSchema(row *sql.Row) (*Schema, error) {
	var sc Schema
	var fieldsJSON []byte
	err := row.Scan(&sc.ID, &sc.Name, &sc.Version, &fieldsJSON, &sc.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(fieldsJSON, &sc.Fields); err != nil {
		return nil, fmt.Errorf("unmarshal fields: %w", err)
	}
	return &sc, nil
}
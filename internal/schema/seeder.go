package schema

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// SeedFromFile reads a JSON seed file and registers any schemas
// that don't already exist in the registry
func SeedFromFile(store *Store, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read seed file: %w", err)
	}

	var schemas []struct {
		Name   string     `json:"name"`
		Fields []FieldDef `json:"fields"`
	}
	if err := json.Unmarshal(data, &schemas); err != nil {
		return fmt.Errorf("parse seed file: %w", err)
	}

	for _, s := range schemas {
		// check if schema already exists
		existing, err := store.GetLatest(s.Name)
		if err == nil {
			log.Printf("seed: schema %q already exists at v%d, skipping", s.Name, existing.Version)
			continue
		}

		sc, err := store.Register(s.Name, s.Fields)
		if err != nil {
			return fmt.Errorf("seed schema %q: %w", s.Name, err)
		}
		log.Printf("seed: registered schema %q v%d with %d fields", sc.Name, sc.Version, len(sc.Fields))
	}

	return nil
}
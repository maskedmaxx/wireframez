package schema

import (
	"testing"
	"os"
)

func getTestConnStr() string {
	conn := os.Getenv("WIREFRAMEZ_DB_URL")
	if conn == "" {
		return "host=127.0.0.1 port=5433 user=wireframez password=wireframez dbname=wireframez sslmode=disable"
	}
	return conn
}

func TestSchemaRegistry(t *testing.T) {
	store, err := NewStore(getTestConnStr())
	if err != nil {
		t.Fatalf("connect to db: %v", err)
	}
	defer store.Close()

	// register a schema
	fields := []FieldDef{
		{Name: "id", Type: "int32"},
		{Name: "name", Type: "string"},
		{Name: "score", Type: "float64"},
		{Name: "active", Type: "bool"},
	}

	sc, err := store.Register("user", fields)
	if err != nil {
		t.Fatalf("register schema: %v", err)
	}
	t.Logf("registered schema: %s v%d (id=%d)", sc.Name, sc.Version, sc.ID)

	// fetch latest
	latest, err := store.GetLatest("user")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	t.Logf("latest schema: %s v%d with %d fields", latest.Name, latest.Version, len(latest.Fields))

	// register a new version
	fields = append(fields, FieldDef{Name: "email", Type: "string"})
	sc2, err := store.Register("user", fields)
	if err != nil {
		t.Fatalf("register v2: %v", err)
	}
	t.Logf("registered new version: %s v%d", sc2.Name, sc2.Version)

	// verify version incremented
	if sc2.Version != sc.Version+1 {
		t.Errorf("expected version %d, got %d", sc.Version+1, sc2.Version)
	}

	// list all schemas
	all, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	t.Logf("total schemas in registry: %d", len(all))
}
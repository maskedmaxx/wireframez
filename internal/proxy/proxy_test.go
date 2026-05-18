package proxy

import (
	"encoding/json"
	"testing"

	"github.com/maskedmaxx/wireframez/internal/schema"
)

func TestJsonToBinaryRoundtrip(t *testing.T) {
	sc := &schema.Schema{
		Name:    "user",
		Version: 1,
		Fields: []schema.FieldDef{
			{Name: "id", Type: "int32"},
			{Name: "name", Type: "string"},
			{Name: "score", Type: "float64"},
			{Name: "active", Type: "bool"},
		},
	}

	original := map[string]any{
		"id":     float64(42),
		"name":   "alice",
		"score":  float64(98.6),
		"active": true,
	}

	jsonBody, _ := json.Marshal(original)
	t.Logf("original JSON:  %d bytes", len(jsonBody))

	binary, err := jsonToBinary(jsonBody, sc)
	if err != nil {
		t.Fatalf("jsonToBinary: %v", err)
	}
	t.Logf("binary encoded: %d bytes", len(binary))

	recovered, err := BinaryToJSON(binary)
	if err != nil {
		t.Fatalf("binaryToJSON: %v", err)
	}
	t.Logf("recovered JSON: %s", string(recovered))

	var result map[string]any
	if err := json.Unmarshal(recovered, &result); err != nil {
		t.Fatalf("unmarshal recovered: %v", err)
	}

	if result["name"] != original["name"] {
		t.Errorf("name mismatch: got %v want %v", result["name"], original["name"])
	}
	if result["active"] != original["active"] {
		t.Errorf("active mismatch: got %v want %v", result["active"], original["active"])
	}

	t.Logf("savings: %d bytes (%.0f%%)",
		len(jsonBody)-len(binary),
		float64(len(jsonBody)-len(binary))/float64(len(jsonBody))*100,
	)
}
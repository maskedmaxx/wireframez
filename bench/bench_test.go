package bench

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/maskedmaxx/wireframez/internal/codec"
	"github.com/maskedmaxx/wireframez/internal/proxy"
	"github.com/maskedmaxx/wireframez/internal/schema"
)

// simulated user record
type User struct {
	ID     int32   `json:"id"`
	Name   string  `json:"name"`
	Email  string  `json:"email"`
	Score  float64 `json:"score"`
	Active bool    `json:"active"`
}

var testSchema = &schema.Schema{
	Name:    "user",
	Version: 1,
	Fields: []schema.FieldDef{
		{Name: "id", Type: "int32"},
		{Name: "name", Type: "string"},
		{Name: "email", Type: "string"},
		{Name: "score", Type: "float64"},
		{Name: "active", Type: "bool"},
	},
}

var sampleUsers = []User{
	{ID: 1, Name: "alice", Email: "alice@example.com", Score: 98.6, Active: true},
	{ID: 2, Name: "bob", Email: "bob@example.com", Score: 87.3, Active: false},
	{ID: 3, Name: "charlie", Email: "charlie@example.com", Score: 91.2, Active: true},
	{ID: 4, Name: "diana", Email: "diana@example.com", Score: 76.5, Active: true},
	{ID: 5, Name: "eve", Email: "eve@example.com", Score: 99.1, Active: false},
}

// --- Size comparison ---

func TestSizeComparison(t *testing.T) {
	fmt.Println("\n=== SIZE COMPARISON ===")
	fmt.Printf("%-20s %-12s %-12s %-10s\n", "Payload", "JSON (bytes)", "Binary (bytes)", "Savings")
	fmt.Println("--------------------------------------------------------------")

	for n := 1; n <= len(sampleUsers); n++ {
		users := sampleUsers[:n]

		// JSON size
		jsonBytes, _ := json.Marshal(users)
		jsonSize := len(jsonBytes)

		// Binary size
		binarySize := 0
		for _, u := range users {
			fields := userToFields(u)
			encoded, _ := codec.Encode(fields)
			binarySize += len(encoded)
		}

		savings := float64(jsonSize-binarySize) / float64(jsonSize) * 100
		fmt.Printf("%-20s %-12d %-12d %.1f%%\n",
			fmt.Sprintf("%d user(s)", n),
			jsonSize,
			binarySize,
			savings,
		)
	}
}

// --- Serialization benchmarks ---

func BenchmarkJSONMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, u := range sampleUsers {
			json.Marshal(u)
		}
	}
}

func BenchmarkBinaryEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, u := range sampleUsers {
			fields := userToFields(u)
			codec.Encode(fields)
		}
	}
}

// --- Deserialization benchmarks ---

func BenchmarkJSONUnmarshal(b *testing.B) {
	jsonBytes, _ := json.Marshal(sampleUsers[0])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var u User
		json.Unmarshal(jsonBytes, &u)
	}
}

func BenchmarkBinaryDecode(b *testing.B) {
	fields := userToFields(sampleUsers[0])
	encoded, _ := codec.Encode(fields)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		codec.Decode(encoded)
	}
}

// --- Roundtrip benchmarks ---

func BenchmarkJSONRoundtrip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(sampleUsers[0])
		var u User
		json.Unmarshal(data, &u)
	}
}

func BenchmarkBinaryRoundtrip(b *testing.B) {
	for i := 0; i < b.N; i++ {
		fields := userToFields(sampleUsers[0])
		encoded, _ := codec.Encode(fields)
		codec.Decode(encoded)
	}
}

// --- Proxy transcoding benchmark ---

func BenchmarkProxyTranscode(b *testing.B) {
	jsonBytes, _ := json.Marshal(sampleUsers[0])
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary, _ := proxy.JSONToBinaryPublic(jsonBytes, testSchema)
		proxy.BinaryToJSON(binary)
	}
}

// helper
func userToFields(u User) []codec.Field {
	return []codec.Field{
		{Name: "id", Type: codec.TypeInt32, Value: u.ID},
		{Name: "name", Type: codec.TypeString, Value: u.Name},
		{Name: "email", Type: codec.TypeString, Value: u.Email},
		{Name: "score", Type: codec.TypeFloat64, Value: u.Score},
		{Name: "active", Type: codec.TypeBool, Value: u.Active},
	}
}
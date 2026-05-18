package codec

import (
	"testing"
	"fmt"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	original := []Field{
		{Name: "id", Type: TypeInt32, Value: int32(42)},
		{Name: "name", Type: TypeString, Value: "alice"},
		{Name: "score", Type: TypeFloat64, Value: float64(98.6)},
		{Name: "active", Type: TypeBool, Value: true},
	}

	encoded, err := Encode(original)
	if err != nil {
		t.Fatalf("encode failed: %v", err)
	}

	fmt.Printf("JSON equivalent would be ~%d bytes\n", jsonSize(original))
	fmt.Printf("Binary encoded:          %d bytes\n", len(encoded))

	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	for i, f := range decoded {
		if f.Name != original[i].Name || f.Value != original[i].Value {
			t.Errorf("field %d mismatch: got %+v, want %+v", i, f, original[i])
		}
	}

	fmt.Println("all fields round-tripped correctly")
}

// rough estimate of what JSON would cost for the same data
func jsonSize(fields []Field) int {
	size := 2 // {}
	for i, f := range fields {
		if i > 0 {
			size++ // comma
		}
		size += len(f.Name) + 3 // "name":
		switch f.Type {
		case TypeInt32, TypeInt64:
			size += 4
		case TypeFloat32, TypeFloat64:
			size += 6
		case TypeBool:
			size += 4
		case TypeString:
			v := f.Value.(string)
			size += len(v) + 2
		}
	}
	return size
}
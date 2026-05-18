package codec

import (
	"encoding/binary"
	"errors"
	"math"
)

// Type tags
const (
	TypeInt32   byte = 0x01
	TypeInt64   byte = 0x02
	TypeFloat32 byte = 0x03
	TypeFloat64 byte = 0x04
	TypeBool    byte = 0x05
	TypeString  byte = 0x06
	TypeNull    byte = 0x07
)

// WireHeader is prepended to every encoded payload
// Layout: [magic: 2 bytes] [version: 2 bytes] [schema_version: 2 bytes]
const (
	MagicByte0 byte = 0x57 // 'W'
	MagicByte1 byte = 0x5A // 'Z'
	WireVersion     = uint16(1)
	HeaderSize      = 6
)

// Field represents a single key-value pair in a payload
type Field struct {
	Name  string
	Type  byte
	Value any
}

// Header contains wire format metadata
type Header struct {
	SchemaVersion uint16
}

// Encode takes a slice of fields and a schema version, returns binary representation
func Encode(fields []Field) ([]byte, error) {
	return EncodeWithVersion(fields, 0)
}

// EncodeWithVersion encodes fields with an explicit schema version in the header
func EncodeWithVersion(fields []Field, schemaVersion uint16) ([]byte, error) {
	// write header
	buf := make([]byte, HeaderSize)
	buf[0] = MagicByte0
	buf[1] = MagicByte1
	binary.BigEndian.PutUint16(buf[2:4], WireVersion)
	binary.BigEndian.PutUint16(buf[4:6], schemaVersion)

	// write field count
	buf = append(buf, byte(len(fields)))

	for _, f := range fields {
		buf = append(buf, f.Type)

		name := []byte(f.Name)
		buf = append(buf, byte(len(name)))
		buf = append(buf, name...)

		encoded, err := encodeValue(f.Type, f.Value)
		if err != nil {
			return nil, err
		}
		buf = append(buf, encoded...)
	}

	return buf, nil
}

// DecodeHeader reads just the wire header without decoding fields
func DecodeHeader(buf []byte) (*Header, error) {
	if len(buf) < HeaderSize {
		return nil, errors.New("buffer too short for header")
	}
	if buf[0] != MagicByte0 || buf[1] != MagicByte1 {
		return nil, errors.New("invalid magic bytes: not a wireframez payload")
	}
	return &Header{
		SchemaVersion: binary.BigEndian.Uint16(buf[4:6]),
	}, nil
}

// Decode takes a binary buffer and returns the fields
func Decode(buf []byte) ([]Field, error) {
	_, fields, err := DecodeWithHeader(buf)
	return fields, err
}

// DecodeWithHeader returns both the header and fields
func DecodeWithHeader(buf []byte) (*Header, []Field, error) {
	if len(buf) < HeaderSize+1 {
		return nil, nil, errors.New("buffer too short")
	}

	header, err := DecodeHeader(buf)
	if err != nil {
		return nil, nil, err
	}

	pos := HeaderSize
	fieldCount := int(buf[pos])
	pos++

	fields := make([]Field, 0, fieldCount)
	for i := 0; i < fieldCount; i++ {
		if pos >= len(buf) {
			return nil, nil, errors.New("unexpected end of buffer")
		}

		typeTag := buf[pos]
		pos++

		nameLen := int(buf[pos])
		pos++
		name := string(buf[pos : pos+nameLen])
		pos += nameLen

		value, bytesRead, err := decodeValue(typeTag, buf[pos:])
		if err != nil {
			return nil, nil, err
		}
		pos += bytesRead

		fields = append(fields, Field{Name: name, Type: typeTag, Value: value})
	}

	return header, fields, nil
}

// IsWireframezPayload checks if a buffer starts with the wireframez magic bytes
func IsWireframezPayload(buf []byte) bool {
	return len(buf) >= 2 && buf[0] == MagicByte0 && buf[1] == MagicByte1
}

func encodeValue(typeTag byte, value any) ([]byte, error) {
	buf := make([]byte, 8)

	switch typeTag {
	case TypeInt32:
		v, ok := value.(int32)
		if !ok {
			return nil, errors.New("expected int32")
		}
		binary.BigEndian.PutUint32(buf[:4], uint32(v))
		return buf[:4], nil

	case TypeInt64:
		v, ok := value.(int64)
		if !ok {
			return nil, errors.New("expected int64")
		}
		binary.BigEndian.PutUint64(buf[:8], uint64(v))
		return buf[:8], nil

	case TypeFloat32:
		v, ok := value.(float32)
		if !ok {
			return nil, errors.New("expected float32")
		}
		binary.BigEndian.PutUint32(buf[:4], math.Float32bits(v))
		return buf[:4], nil

	case TypeFloat64:
		v, ok := value.(float64)
		if !ok {
			return nil, errors.New("expected float64")
		}
		binary.BigEndian.PutUint64(buf[:8], math.Float64bits(v))
		return buf[:8], nil

	case TypeBool:
		v, ok := value.(bool)
		if !ok {
			return nil, errors.New("expected bool")
		}
		if v {
			return []byte{0x01}, nil
		}
		return []byte{0x00}, nil

	case TypeString:
		v, ok := value.(string)
		if !ok {
			return nil, errors.New("expected string")
		}
		strBytes := []byte(v)
		lenBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBuf, uint32(len(strBytes)))
		return append(lenBuf, strBytes...), nil

	case TypeNull:
		return []byte{}, nil

	default:
		return nil, errors.New("unknown type tag")
	}
}

func decodeValue(typeTag byte, buf []byte) (any, int, error) {
	switch typeTag {
	case TypeInt32:
		v := int32(binary.BigEndian.Uint32(buf[:4]))
		return v, 4, nil

	case TypeInt64:
		v := int64(binary.BigEndian.Uint64(buf[:8]))
		return v, 8, nil

	case TypeFloat32:
		v := math.Float32frombits(binary.BigEndian.Uint32(buf[:4]))
		return v, 4, nil

	case TypeFloat64:
		v := math.Float64frombits(binary.BigEndian.Uint64(buf[:8]))
		return v, 8, nil

	case TypeBool:
		return buf[0] == 0x01, 1, nil

	case TypeString:
		strLen := int(binary.BigEndian.Uint32(buf[:4]))
		v := string(buf[4 : 4+strLen])
		return v, 4 + strLen, nil

	case TypeNull:
		return nil, 0, nil

	default:
		return nil, 0, errors.New("unknown type tag")
	}
}
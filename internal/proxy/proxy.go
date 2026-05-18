package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maskedmaxx/wireframez/internal/codec"
	"github.com/maskedmaxx/wireframez/internal/metrics"
	"github.com/maskedmaxx/wireframez/internal/schema"
)

// Target represents a registered backend
type Target struct {
	SchemaName string `json:"schema_name"`
	URL        string `json:"url"`
}

// Proxy is a transcoding reverse proxy with dynamic target registration
type Proxy struct {
	store   *schema.Store
	targets map[string]*url.URL
	mu      sync.RWMutex
}

// NewProxy creates a new transcoding proxy
func NewProxy(store *schema.Store) *Proxy {
	return &Proxy{
		store:   store,
		targets: make(map[string]*url.URL),
	}
}

// RegisterTarget registers a backend target for a schema
func (p *Proxy) RegisterTarget(schemaName, targetURL string) error {
	u, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("parse target url: %w", err)
	}
	p.mu.Lock()
	p.targets[schemaName] = u
	p.mu.Unlock()
	return nil
}

// DeregisterTarget removes a backend target
func (p *Proxy) DeregisterTarget(schemaName string) {
	p.mu.Lock()
	delete(p.targets, schemaName)
	p.mu.Unlock()
}

// ListTargets returns all registered targets
func (p *Proxy) ListTargets() []Target {
	p.mu.RLock()
	defer p.mu.RUnlock()
	targets := make([]Target, 0, len(p.targets))
	for name, u := range p.targets {
		targets = append(targets, Target{SchemaName: name, URL: u.String()})
	}
	return targets
}

// ServeHTTP handles incoming proxy requests
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metrics.ActiveConnections.Inc()
	defer metrics.ActiveConnections.Dec()

	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "missing schema name in path", http.StatusBadRequest)
		metrics.RequestsTotal.WithLabelValues("unknown", "400").Inc()
		return
	}
	schemaName := parts[0]

	// read body first so we can use it for both passthrough and transcoding
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "read body failed", http.StatusInternalServerError)
		metrics.RequestsTotal.WithLabelValues(schemaName, "500").Inc()
		return
	}
	defer r.Body.Close()

	// look up schema — if not found, passthrough mode
	schemaStart := time.Now()
	sc, schemaErr := p.store.GetLatest(schemaName)
	metrics.SchemaLookupDuration.Observe(time.Since(schemaStart).Seconds())

	// find target
	p.mu.RLock()
	target, hasTarget := p.targets[schemaName]
	p.mu.RUnlock()

	if !hasTarget {
		http.Error(w, fmt.Sprintf("no target registered for schema %q", schemaName), http.StatusBadGateway)
		metrics.RequestsTotal.WithLabelValues(schemaName, "502").Inc()
		return
	}

	var forwardBody []byte

	if schemaErr != nil {
		// passthrough — no schema registered, forward JSON as-is
		forwardBody = body
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Wireframez-Mode", "passthrough")
	} else {
		// transcode JSON -> binary
		if len(body) > 0 {
			metrics.PayloadSizeJSON.WithLabelValues(schemaName).Observe(float64(len(body)))

			transcodeStart := time.Now()
			transcoded, err := jsonToBinary(body, sc)
			metrics.TranscodingDuration.WithLabelValues(schemaName).Observe(time.Since(transcodeStart).Seconds())

			if err != nil {
				// transcode failed — fallback to passthrough
				forwardBody = body
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("X-Wireframez-Mode", "passthrough-fallback")
			} else {
				forwardBody = transcoded
				metrics.PayloadSizeBinary.WithLabelValues(schemaName).Observe(float64(len(transcoded)))
				metrics.CompressionRatio.WithLabelValues(schemaName).Observe(
					float64(len(transcoded)) / float64(len(body)),
				)
				r.Header.Set("Content-Type", "application/octet-stream")
				r.Header.Set("X-Wireframez-Mode", "transcoded")
				r.Header.Set("X-Wireframez-Schema", schemaName)
				r.Header.Set("X-Wireframez-Version", strconv.Itoa(sc.Version))
			}
		}
	}

	reverseProxy := httputil.NewSingleHostReverseProxy(target)
	r.Body = io.NopCloser(bytes.NewReader(forwardBody))
	r.ContentLength = int64(len(forwardBody))

	metrics.RequestsTotal.WithLabelValues(schemaName, "200").Inc()
	reverseProxy.ServeHTTP(w, r)
}

// jsonToBinary converts a JSON payload to wireframez binary format
func jsonToBinary(jsonBody []byte, sc *schema.Schema) ([]byte, error) {
	var raw map[string]any
	if err := json.Unmarshal(jsonBody, &raw); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	fields := make([]codec.Field, 0, len(sc.Fields))
	for _, fd := range sc.Fields {
		val, ok := raw[fd.Name]
		if !ok {
			continue
		}

		typeTag, err := typeTagFromString(fd.Type)
		if err != nil {
			return nil, err
		}

		val, err = coerceValue(val, fd.Type)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", fd.Name, err)
		}

		fields = append(fields, codec.Field{
			Name:  fd.Name,
			Type:  typeTag,
			Value: val,
		})
	}

	// embed schema version in the wire header
	return codec.EncodeWithVersion(fields, uint16(sc.Version))
}

// BinaryToJSON converts wireframez binary back to JSON
func BinaryToJSON(data []byte) ([]byte, error) {
	// validate magic bytes first
	if !codec.IsWireframezPayload(data) {
		return nil, fmt.Errorf("not a wireframez payload")
	}

	header, fields, err := codec.DecodeWithHeader(data)
	if err != nil {
		return nil, err
	}

	out := make(map[string]any, len(fields)+1)
	out["_schema_version"] = header.SchemaVersion
	for _, f := range fields {
		out[f.Name] = f.Value
	}

	return json.Marshal(out)
}

// JSONToBinaryPublic is the exported version for benchmarking
func JSONToBinaryPublic(jsonBody []byte, sc *schema.Schema) ([]byte, error) {
	return jsonToBinary(jsonBody, sc)
}

func typeTagFromString(t string) (byte, error) {
	switch t {
	case "int32":
		return codec.TypeInt32, nil
	case "int64":
		return codec.TypeInt64, nil
	case "float32":
		return codec.TypeFloat32, nil
	case "float64":
		return codec.TypeFloat64, nil
	case "bool":
		return codec.TypeBool, nil
	case "string":
		return codec.TypeString, nil
	default:
		return 0, fmt.Errorf("unknown type %q", t)
	}
}

func coerceValue(val any, typeName string) (any, error) {
	switch typeName {
	case "int32":
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected number")
		}
		return int32(f), nil
	case "int64":
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected number")
		}
		return int64(f), nil
	case "float32":
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected number")
		}
		return float32(f), nil
	case "float64":
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected number")
		}
		return f, nil
	case "bool":
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool")
		}
		return b, nil
	case "string":
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string")
		}
		return s, nil
	}
	return nil, fmt.Errorf("unknown type %q", typeName)
}
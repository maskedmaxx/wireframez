package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/maskedmaxx/wireframez/internal/proxy"
	"github.com/maskedmaxx/wireframez/internal/schema"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type registerTargetRequest struct {
	SchemaName string `json:"schema_name"`
	URL        string `json:"url"`
}

func main() {
	connStr := os.Getenv("WIREFRAMEZ_DB_URL")
	if connStr == "" {
		connStr = "host=127.0.0.1 port=5433 user=wireframez password=wireframez dbname=wireframez sslmode=disable"
	}

	store, err := schema.NewStore(connStr)
	if err != nil {
		log.Fatalf("connect to db: %v", err)
	}
	defer store.Close()

    // seed schemas from file if provided
    seedPath := os.Getenv("WIREFRAMEZ_SEED_FILE")
    if seedPath != "" {
        if err := schema.SeedFromFile(store, seedPath); err != nil {
            log.Printf("warning: seed failed: %v", err)
        }
    }

	p := proxy.NewProxy(store)

	mux := http.NewServeMux()

	// metrics
	mux.Handle("/metrics", promhttp.Handler())

	// target management API
	mux.HandleFunc("/targets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			var req registerTargetRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			if err := p.RegisterTarget(req.SchemaName, req.URL); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{
				"status":      "registered",
				"schema_name": req.SchemaName,
				"url":         req.URL,
			})

		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(p.ListTargets())

		case http.MethodDelete:
			var req registerTargetRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			p.DeregisterTarget(req.SchemaName)
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// all other requests go to the proxy
	mux.Handle("/", p)

	addr := ":8080"
	fmt.Printf("wireframez proxy listening on %s\n", addr)
	fmt.Printf("targets API:   http://localhost%s/targets\n", addr)
	fmt.Printf("metrics:       http://localhost%s/metrics\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
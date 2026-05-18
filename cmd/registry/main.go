package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/maskedmaxx/wireframez/internal/schema"
)

type registerRequest struct {
	Name   string           `json:"name"`
	Fields []schema.FieldDef `json:"fields"`
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

	mux := http.NewServeMux()

	// POST /schemas — register a new schema
	mux.HandleFunc("/schemas", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid body: %v", err), http.StatusBadRequest)
			return
		}

		if req.Name == "" || len(req.Fields) == 0 {
			http.Error(w, "name and fields required", http.StatusBadRequest)
			return
		}

		sc, err := store.Register(req.Name, req.Fields)
		if err != nil {
			http.Error(w, fmt.Sprintf("register failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sc)
	})

	// GET /schemas — list all schemas
	mux.HandleFunc("/schemas/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// /schemas/<name> or /schemas/<name>/<version>
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/schemas/"), "/")
		name := parts[0]

		if name == "" {
			// list all
			all, err := store.List()
			if err != nil {
				http.Error(w, fmt.Sprintf("list failed: %v", err), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(all)
			return
		}

		if len(parts) == 2 && parts[1] != "" {
			// specific version
			version, err := strconv.Atoi(parts[1])
			if err != nil {
				http.Error(w, "invalid version", http.StatusBadRequest)
				return
			}
			sc, err := store.GetVersion(name, version)
			if err != nil {
				http.Error(w, fmt.Sprintf("not found: %v", err), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(sc)
			return
		}

		// latest version
		sc, err := store.GetLatest(name)
		if err != nil {
			http.Error(w, fmt.Sprintf("not found: %v", err), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sc)
	})

	addr := ":8081"
	fmt.Printf("wireframez registry listening on %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server: %v", err)
	}
}
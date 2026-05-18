package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/maskedmaxx/wireframez/internal/proxy"
	"github.com/maskedmaxx/wireframez/internal/schema"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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

	p := proxy.NewProxy(store)

	if err := p.RegisterTarget("user", "http://localhost:9090"); err != nil {
		log.Fatalf("register target: %v", err)
	}

	// metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// proxy handler
	http.Handle("/", p)

	addr := ":8080"
	fmt.Printf("wireframez proxy listening on %s\n", addr)
	fmt.Printf("metrics available at http://localhost%s/metrics\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server: %v", err)
	}
}
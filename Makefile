.PHONY: help up down build test bench lint clean seed logs

# default target
help:
	@echo ""
	@echo "  wireframez - binary protocol gateway"
	@echo ""
	@echo "  make up       start the full stack (proxy, registry, postgres, prometheus, grafana)"
	@echo "  make down     stop and remove containers"
	@echo "  make reset    stop, wipe volumes, and restart fresh"
	@echo "  make build    build docker images"
	@echo "  make test     run all tests"
	@echo "  make bench    run benchmarks vs JSON"
	@echo "  make lint     run go vet"
	@echo "  make seed     register default schemas via API"
	@echo "  make traffic  send 100 test requests through the proxy"
	@echo "  make logs     tail all container logs"
	@echo "  make clean    remove build artifacts"
	@echo ""

up:
	docker-compose up --build -d
	@echo ""
	@echo "  proxy:      http://localhost:8080"
	@echo "  registry:   http://localhost:8081"
	@echo "  grafana:    http://localhost:3000  (admin / wireframez)"
	@echo "  prometheus: http://localhost:9090"
	@echo ""

down:
	docker-compose down

reset:
	docker-compose down -v
	docker-compose up --build -d

build:
	docker-compose build

test:
	go test ./internal/... -v

bench:
	go test ./bench/ -bench=. -benchmem -v

lint:
	go vet ./...

seed:
	@echo "registering user schema..."
	@curl -s -X POST http://localhost:8081/schemas \
		-H "Content-Type: application/json" \
		-d '{"name":"user","fields":[{"name":"id","type":"int32"},{"name":"name","type":"string"},{"name":"email","type":"string"},{"name":"score","type":"float64"},{"name":"active","type":"bool"}]}' | jq .
	@echo "registering order schema..."
	@curl -s -X POST http://localhost:8081/schemas \
		-H "Content-Type: application/json" \
		-d '{"name":"order","fields":[{"name":"id","type":"int64"},{"name":"user_id","type":"int32"},{"name":"total","type":"float64"},{"name":"status","type":"string"},{"name":"paid","type":"bool"}]}' | jq .

traffic:
	@echo "sending 100 requests through proxy..."
	@for i in $$(seq 1 100); do \
		curl -s -X POST http://localhost:8080/user \
			-H "Content-Type: application/json" \
			-d '{"id":1,"name":"alice","email":"alice@example.com","score":98.6,"active":true}' \
			> /dev/null; \
	done
	@echo "done. check grafana at http://localhost:3000"

logs:
	docker-compose logs -f

clean:
	rm -f wireframez wireframez-registry
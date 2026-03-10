.PHONY: all build run clean docker-up docker-down docker-logs dev

BINARY_NAME=daemon
BIN_DIR=bin

all: build

build:
	go mod download
	go build -o $(BIN_DIR)/$(BINARY_NAME) cmd/daemon/main.go

run: build
	sudo ./$(BIN_DIR)/$(BINARY_NAME)

clean:
	rm -rf $(BIN_DIR)/
	go clean

docker-up:
	docker-compose up -d
	@echo "✅ Servicios iniciados:"
	@echo "   - Valkey: localhost:6379"
	@echo "   - Grafana: http://localhost:3000 (admin/admin)"

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

dev: docker-up run

docker-reset: docker-down docker-up

.PHONY: test deps
test:
	go test ./...

deps:
	go get -u ./...
	go mod tidy
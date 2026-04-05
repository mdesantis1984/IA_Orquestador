.PHONY: all build build-linux build-postgres build-all test run run-http docker-build docker-push deploy-k8s clean install deps

# Variables
BINARY_NAME=orchestrator
BUILD_DIR=./bin
MAIN_PATH=./cmd/orchestrator
DB_PATH=./orchestrator.db
IMAGE?=ghcr.io/your-username/ia-orquestador
TAG?=latest

all: deps build

deps:
	@echo "→ Downloading dependencies..."
	go mod download
	go mod verify

build:
	@echo "→ Building $(BINARY_NAME) (sqlite, $(shell go env GOARCH))..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-linux:
	@echo "→ Building $(BINARY_NAME) for Linux/amd64 (sqlite)..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "✓ Linux binary: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

build-postgres:
	@echo "→ Building $(BINARY_NAME) with Postgres driver..."
	mkdir -p $(BUILD_DIR)
	go build -tags postgres -o $(BUILD_DIR)/$(BINARY_NAME)-postgres $(MAIN_PATH)
	@echo "✓ Postgres binary: $(BUILD_DIR)/$(BINARY_NAME)-postgres"

build-postgres-linux:
	@echo "→ Building $(BINARY_NAME) for Linux/amd64 (postgres)..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -tags postgres -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-postgres $(MAIN_PATH)
	@echo "✓ Postgres Linux binary: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64-postgres"

build-all: build build-linux build-postgres build-postgres-linux

test:
	@echo "→ Running tests..."
	go test -v -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

test-coverage:
	@echo "→ Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

run: build
	@echo "→ Running $(BINARY_NAME) (STDIO mode)..."
	$(BUILD_DIR)/$(BINARY_NAME) -transport=stdio -db=$(DB_PATH)

run-http: build
	@echo "→ Running $(BINARY_NAME) (HTTP mode)..."
	$(BUILD_DIR)/$(BINARY_NAME) -transport=http -http-addr=:8080 -db=$(DB_PATH)

run-http-postgres: build-postgres
	@echo "→ Running $(BINARY_NAME) (HTTP + Postgres)..."
	$(BUILD_DIR)/$(BINARY_NAME)-postgres -transport=http -http-addr=:8080 \
		-db-driver=postgres -db-dsn="$(DATABASE_URL)"

docker-build:
	@echo "→ Building Docker image $(IMAGE):$(TAG)..."
	docker build -t $(IMAGE):$(TAG) .
	@echo "✓ Image built: $(IMAGE):$(TAG)"

docker-push:
	@echo "→ Pushing $(IMAGE):$(TAG)..."
	docker push $(IMAGE):$(TAG)

deploy-k8s:
	@echo "→ Applying Kubernetes manifests..."
	kubectl apply -k deploy/kubernetes/
	@echo "✓ Deployed"

install: build
	@echo "→ Installing to /opt/ia-orquestador/..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /opt/ia-orquestador/
	@echo "✓ Installed"

clean:
	@echo "→ Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(DB_PATH)
	rm -f coverage.out coverage.html
	@echo "✓ Clean complete"

fmt:
	@echo "→ Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

lint:
	@echo "→ Running linter..."
	golangci-lint run ./...
	@echo "✓ Lint complete"

db-reset:
	@echo "→ Resetting database..."
	rm -f $(DB_PATH)
	@echo "✓ Database reset"

help:
	@echo "Available targets:"
	@echo "  all                   - Download deps and build"
	@echo "  deps                  - Download dependencies"
	@echo "  build                 - Build (SQLite, local arch)"
	@echo "  build-linux           - Build for Linux/amd64 (SQLite)"
	@echo "  build-postgres        - Build with PostgreSQL driver (local)"
	@echo "  build-postgres-linux  - Build for Linux/amd64 (PostgreSQL)"
	@echo "  build-all             - All binaries (all combos)"
	@echo "  test                  - Run unit tests"
	@echo "  test-coverage         - Generate coverage report"
	@echo "  run                   - Build and run (STDIO+SQLite)"
	@echo "  run-http              - Build and run (HTTP+SQLite)"
	@echo "  run-http-postgres     - Build and run (HTTP+Postgres, needs DATABASE_URL)"
	@echo "  docker-build          - Build Docker image (IMAGE/TAG vars)"
	@echo "  docker-push           - Push Docker image"
	@echo "  deploy-k8s            - kubectl apply -k deploy/kubernetes/"
	@echo "  install               - Install to /opt/ia-orquestador/"
	@echo "  clean                 - Remove build artifacts"
	@echo "  fmt                   - Format code"
	@echo "  lint                  - Run linter"
	@echo "  db-reset      - Reset database"
	@echo "  help          - Show this help"

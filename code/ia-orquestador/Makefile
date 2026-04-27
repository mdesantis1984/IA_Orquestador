.PHONY: all build build-linux build-all test test-coverage run run-http docker-build docker-push deploy-k8s clean install deps fmt lint help

# Variables
BINARY_NAME=orchestrator
BUILD_DIR=./bin
MAIN_PATH=./cmd/orchestrator
IMAGE?=ghcr.io/your-username/ia-orquestador
TAG?=latest

all: deps build

deps:
	@echo "→ Downloading dependencies..."
	go mod download
	go mod verify

build:
	@echo "→ Building $(BINARY_NAME) ($(shell go env GOARCH))..."
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✓ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-linux:
	@echo "→ Building $(BINARY_NAME) for Linux/amd64..."
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "✓ Linux binary: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

build-all: build build-linux

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
	@echo "→ Running $(BINARY_NAME) (STDIO mode, requires DATABASE_URL)..."
	$(BUILD_DIR)/$(BINARY_NAME) \
		-transport=stdio \
		-db-dsn="$(DATABASE_URL)"

run-http: build
	@echo "→ Running $(BINARY_NAME) (HTTP mode, requires DATABASE_URL)..."
	$(BUILD_DIR)/$(BINARY_NAME) \
		-transport=http \
		-http-addr=:8080 \
		-db-dsn="$(DATABASE_URL)"

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

help:
	@echo "Available targets:"
	@echo "  all              - Download deps and build"
	@echo "  deps             - Download dependencies"
	@echo "  build            - Build for local arch"
	@echo "  build-linux      - Cross-compile for Linux/amd64"
	@echo "  build-all        - Build local + Linux"
	@echo "  test             - Run unit tests"
	@echo "  test-coverage    - Generate coverage report"
	@echo "  run              - Build and run (STDIO, needs DATABASE_URL)"
	@echo "  run-http         - Build and run (HTTP :8080, needs DATABASE_URL)"
	@echo "  docker-build     - Build Docker image (IMAGE/TAG vars)"
	@echo "  docker-push      - Push Docker image"
	@echo "  deploy-k8s       - kubectl apply -k deploy/kubernetes/"
	@echo "  install          - Install to /opt/ia-orquestador/"
	@echo "  clean            - Remove build artifacts"
	@echo "  fmt              - Format code"
	@echo "  lint             - Run linter"
	@echo "  help             - Show this help"

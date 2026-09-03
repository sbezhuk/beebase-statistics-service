.PHONY: run build fmt vet test lint tidy

APP_NAME := server
BIN_DIR  := bin

run: ## Run the service locally (loads .env).
	go run ./cmd/server

build: ## Build the service binary into bin/.
	CGO_ENABLED=0 go build -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

fmt: ## Format all Go source.
	go fmt ./...

vet: ## Run go vet on all packages.
	go vet ./...

test: ## Run the unit test suite.
	go test ./... -v

lint: ## Run golangci-lint, if installed.
	golangci-lint run

tidy: ## Sync go.mod/go.sum with imports.
	go mod tidy

.PHONY: all build build-api build-web run dev clean deps test

# Default target
all: build

# Build everything
build: build-web build-api

# Build the Go API server
build-api:
	@echo "Building API server..."
	@mkdir -p cmd/ccc-api/static
	@if [ -d web/dist ]; then \
		cp -r web/dist/* cmd/ccc-api/static/; \
	fi
	CGO_ENABLED=1 go build -o bin/ccc-api ./cmd/ccc-api

# Build the web frontend
build-web:
	@echo "Building web frontend..."
	cd web && npm install && npm run build

# Run the API server
run: build
	./bin/ccc-api

# Run in development mode (API only, no embedded static)
dev:
	@echo "Starting API server in dev mode..."
	@echo "For frontend dev, run 'cd web && npm run dev' in another terminal"
	CGO_ENABLED=1 go run ./cmd/ccc-api

# Install dependencies
deps:
	go mod tidy
	cd web && npm install

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -rf web/node_modules/
	rm -rf cmd/ccc-api/static/*
	rm -f ccc.db

# Quick API build (without frontend)
build-api-only:
	CGO_ENABLED=1 go build -o bin/ccc-api ./cmd/ccc-api

# Help
help:
	@echo "Available targets:"
	@echo "  make build      - Build both API and web frontend"
	@echo "  make build-api  - Build only the API server"
	@echo "  make build-web  - Build only the web frontend"
	@echo "  make run        - Build and run the server"
	@echo "  make dev        - Run API in dev mode (hot reload frontend separately)"
	@echo "  make deps       - Install all dependencies"
	@echo "  make test       - Run tests"
	@echo "  make clean      - Clean all build artifacts"

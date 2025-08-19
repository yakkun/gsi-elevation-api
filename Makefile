.PHONY: all build test bench install clean run dev lint vet fmt tidy help

BINARY_NAME=elevation-api
INSTALL_DIR=/opt/elevation-api
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-s -w"

all: clean build test

build:
	@echo "Building $(BINARY_NAME)..."
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/server

test:
	@echo "Running tests..."
	$(GO) test $(GOFLAGS) -race -coverprofile=coverage.out ./...
	@echo "Coverage report:"
	@$(GO) tool cover -func=coverage.out | tail -1

bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem -run=^$$ ./internal/elevation/

install: build
	@echo "Installing to $(INSTALL_DIR)..."
	@sudo mkdir -p $(INSTALL_DIR)
	@sudo mkdir -p $(INSTALL_DIR)/data
	@sudo mkdir -p $(INSTALL_DIR)/config
	@sudo cp $(BINARY_NAME) $(INSTALL_DIR)/
	@sudo cp -r config/* $(INSTALL_DIR)/config/
	@if [ -d "data" ] && [ "$(ls -A data)" ]; then \
		sudo cp -r data/* $(INSTALL_DIR)/data/; \
	fi
	@sudo chmod 755 $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "Installation complete"

clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out
	@rm -f cpu.prof mem.prof
	@$(GO) clean

run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

dev:
	@echo "Running in development mode with auto-reload..."
	@command -v air > /dev/null 2>&1 || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	@air

lint:
	@echo "Running linter..."
	@command -v golangci-lint > /dev/null 2>&1 || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run ./...

vet:
	@echo "Running go vet..."
	@$(GO) vet ./...

fmt:
	@echo "Formatting code..."
	@$(GO) fmt ./...

tidy:
	@echo "Tidying module dependencies..."
	@$(GO) mod tidy

generate-test-data:
	@echo "Generating test elevation data..."
	@mkdir -p data
	@python3 scripts/convert_data.py --test --output data/elevation.bin --header data/elevation.bin.header

prof-cpu:
	@echo "Running CPU profiling..."
	@$(GO) test -cpuprofile=cpu.prof -bench=. -run=^$$ ./internal/elevation/
	@$(GO) tool pprof -http=:8081 cpu.prof

prof-mem:
	@echo "Running memory profiling..."
	@$(GO) test -memprofile=mem.prof -bench=. -run=^$$ ./internal/elevation/
	@$(GO) tool pprof -http=:8081 mem.prof

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):latest .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 $(BINARY_NAME):latest

help:
	@echo "Available targets:"
	@echo "  make build              - Build the binary"
	@echo "  make test               - Run tests with coverage"
	@echo "  make bench              - Run benchmarks"
	@echo "  make install            - Install to system (requires sudo)"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make run                - Build and run the server"
	@echo "  make dev                - Run with auto-reload (requires air)"
	@echo "  make lint               - Run linter"
	@echo "  make vet                - Run go vet"
	@echo "  make fmt                - Format code"
	@echo "  make tidy               - Tidy module dependencies"
	@echo "  make generate-test-data - Generate test elevation data"
	@echo "  make prof-cpu           - Run CPU profiling"
	@echo "  make prof-mem           - Run memory profiling"
	@echo "  make docker-build       - Build Docker image"
	@echo "  make docker-run         - Run Docker container"
	@echo "  make help               - Show this help message"
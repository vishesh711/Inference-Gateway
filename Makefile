.PHONY: all build run clean test fmt deps benchmark help

# Default target
all: deps build

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	@go mod download
	@go mod tidy

# Build both binaries
build:
	@echo "Building gateway..."
	@mkdir -p bin
	@go build -o bin/gateway ./cmd/gateway
	@echo "Building load generator..."
	@go build -o bin/loadgen ./cmd/loadgen
	@echo "Build complete. Binaries in ./bin/"

# Run the gateway
run:
	@echo "Starting gateway on port 8000..."
	@./bin/gateway -config config.yaml

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f config.yaml.bak

# Run concurrency benchmark sweep
benchmark: build
	@echo "Running concurrency benchmark sweep..."
	@mkdir -p results
	@for c in 1 2 4 8 16 32; do \
		echo ""; \
		echo "======================================"; \
		echo "Testing concurrency: $$c"; \
		echo "======================================"; \
		sed -i.bak "s/max_in_flight: .*/max_in_flight: $$c/" config.yaml; \
		pkill -9 gateway 2>/dev/null || true; \
		./bin/gateway -config config.yaml > /dev/null 2>&1 & \
		GATEWAY_PID=$$!; \
		sleep 5; \
		./bin/loadgen -workers $$c -duration 3m -warmup 30s | tee results/concurrency_$$c.txt; \
		kill $$GATEWAY_PID 2>/dev/null || true; \
		sleep 2; \
	done
	@mv config.yaml.bak config.yaml
	@echo ""
	@echo "Benchmark complete. Results in ./results/"

# Quick load test (30 seconds, 4 workers)
load-test: build
	@echo "Running quick load test..."
	@./bin/loadgen -workers 4 -duration 30s -warmup 5s

# Help
help:
	@echo "Inference Gateway - Makefile commands:"
	@echo ""
	@echo "  make deps        - Install Go dependencies"
	@echo "  make build       - Build gateway and loadgen binaries"
	@echo "  make run         - Start the gateway service"
	@echo "  make test        - Run tests"
	@echo "  make fmt         - Format Go code"
	@echo "  make clean       - Remove build artifacts"
	@echo "  make benchmark   - Run full concurrency sweep (takes ~20min)"
	@echo "  make load-test   - Run quick 30s load test"
	@echo "  make all         - Install deps and build (default)"
	@echo "  make help        - Show this help message"
	@echo ""

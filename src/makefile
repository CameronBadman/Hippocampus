.PHONY: build run test clean install fmt lint vet deps help

BINARY_NAME=hippocampus
BINARY_FILE=tree.bin
GO=go
GOFLAGS=-v

build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

test:
	$(GO) test $(GOFLAGS) ./...

test-verbose:
	$(GO) test -v -race ./...

test-cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

clean:
	$(GO) clean
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

install:
	$(GO) mod download
	$(GO) mod verify

deps:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

vet:
	$(GO) vet ./...

help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  run          - Build and run the binary"
	@echo "  test         - Run tests"
	@echo "  test-verbose - Run tests with race detector"
	@echo "  test-cover   - Generate test coverage report"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Download and verify dependencies"
	@echo "  deps         - Tidy dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  vet          - Run go vet"

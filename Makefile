# Makefile for url-shortener protobuf and project management

GOPATH ?= $(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)

PROTOC ?= protoc
PROTOC_GEN_GO ?= protoc-gen-go

PROTO_DIRS ?= proto pb
PROTO_FILES ?= $(shell find $(PROTO_DIRS) -type f -name '*.proto' 2>/dev/null)
OUT_DIR ?= pb

.PHONY: all proto pb clean clean-proto check-tools build test help

all: proto

## check-tools: Check if protoc and protoc-gen-go are installed
check-tools:
	@which $(PROTOC) > /dev/null 2>&1 || (echo "Error: $(PROTOC) is not installed. Please install protobuf compiler (e.g., apt install protobuf-compiler)." && exit 1)
	@which $(PROTOC_GEN_GO) > /dev/null 2>&1 || (echo "Error: $(PROTOC_GEN_GO) is not installed. Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" && exit 1)

## proto: Compile .proto files into Go code (target pb)
proto: check-tools
	@if [ -z "$(PROTO_FILES)" ]; then \
		echo "No .proto files found in $(PROTO_DIRS)"; \
		exit 1; \
	fi
	@mkdir -p $(OUT_DIR)
	@echo "Compiling proto files: $(PROTO_FILES)..."
	$(PROTOC) --proto_path=. $(addprefix -I,$(PROTO_DIRS)) --go_out=. $(PROTO_FILES)
	@echo "Successfully compiled proto files to $(OUT_DIR)/"

## pb: Alias for proto
pb: proto

## clean-proto: Remove generated pb.go files
clean-proto:
	@echo "Removing generated protobuf Go files..."
	@rm -f $(OUT_DIR)/*.pb.go
	@echo "Cleaned generated files in $(OUT_DIR)/"

## clean: Remove build artifacts and generated files
clean: clean-proto

## build: Build all Go packages
build: proto
	@echo "Building Go packages..."
	go build ./...

## test: Run tests
test: proto
	@echo "Running tests..."
	go test ./...

## help: Display this help screen
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  proto / pb   Compile .proto files to Go code in $(OUT_DIR)/"
	@echo "  clean-proto  Remove generated .pb.go files"
	@echo "  build        Compile proto files and build Go binaries"
	@echo "  test         Compile proto files and run Go tests"
	@echo "  clean        Remove all generated and build artifacts"
	@echo "  check-tools  Check if protoc and protoc-gen-go are installed"
	@echo "  help         Show this help message"

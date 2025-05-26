.PHONY: proto clean build test help

# Generate protobuf files
proto:
	@echo "Generating protobuf files..."
	buf generate proto
	@echo "Protobuf files generated successfully!"

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf dtvm/dtvm.pb.go dtvm/dtvm_grpc.pb.go
	@echo "Clean completed!"

# Build the project
build:
	@echo "Building SEV-SNP server..."
	go build -o bin/sev_snp_server cmd/sev_snp_server/main.go
	@echo "Build completed!"

# Run tests
test:
	@echo "Running tests..."
	go test ./...
	@echo "Tests completed!"

# Format proto files
proto-format:
	@echo "Formatting proto files..."
	buf format -w proto
	@echo "Proto files formatted!"

# Lint proto files
proto-lint:
	@echo "Linting proto files..."
	buf lint proto
	@echo "Proto lint completed!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download
	@echo "Dependencies installed!"

# Help
help:
	@echo "Available commands:"
	@echo "  proto        - Generate protobuf files"
	@echo "  clean        - Clean generated files"
	@echo "  build        - Build the project"
	@echo "  test         - Run tests"
	@echo "  proto-format - Format proto files"
	@echo "  proto-lint   - Lint proto files"
	@echo "  deps         - Install dependencies"
	@echo "  help         - Show this help message" 
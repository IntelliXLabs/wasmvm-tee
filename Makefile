.PHONY: proto clean build test help

# Generate protobuf files including grpc-gateway
proto:
	@echo "Generating protobuf files with buf..."
	buf generate
	@echo "Moving generated files to correct location..."
	@if [ -f proto/dtvm/dtvm.pb.go ]; then mv proto/dtvm/*.pb.go dtvm/; fi
	@if [ -f proto/dtvm/dtvm.pb.gw.go ]; then mv proto/dtvm/*.pb.gw.go dtvm/; fi
	@if [ -f proto/dtvm/dtvm.swagger.json ]; then mv proto/dtvm/*.swagger.json dtvm/; fi
	@echo "Protobuf files generated successfully!"

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	rm -rf dtvm/dtvm.pb.go dtvm/dtvm_grpc.pb.go
	@echo "Clean completed!"

# Build the project
build:
	@echo "Building SEV-SNP server..."
	CGO_ENABLED=1 go build -o bin/sev_snp_server cmd/sev_snp_server/main.go
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

lint-imports:
	@find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*" | while read -r file; do \
		goimports-reviser -company-prefixes github.com/IntelliXLabs -rm-unused -format "$$file"; \
	done
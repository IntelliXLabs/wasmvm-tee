# wasmvm-tee: Trusted Execution Environment for WebAssembly

A secure gRPC service that executes WebAssembly (WASM) bytecode within a Trusted Execution Environment (TEE) using AMD SEV-SNP technology. This project leverages WasmEdge runtime to provide cryptographic attestation of execution results, ensuring integrity and confidentiality of computations with native WASI support.

## Architecture

### Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   gRPC Client   │───▶│  wasmvm-tee     │───▶│   AMD SEV-SNP   │
│                 │    │  Server         │    │   Attestation   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │  WasmEdge       │
                       │  Runtime        │
                       │  + WASI Support │
                       └─────────────────┘
```

### Components

1. **gRPC Server** (`cmd/sev_snp_server/main.go`)
   - Listens for WebAssembly execution requests
   - Configurable port via command line arguments
   - Supports gRPC reflection for debugging

2. **WASMVM TEE Service** (`wasm/server.go`)
   - Implements `WASMVMTeeService` gRPC interface
   - Handles WASM bytecode execution in secure environment
   - Generates cryptographic attestations
   - Powered by WasmEdge runtime with WASI capabilities

3. **WasmEdge Runtime Integration** (`wasm/wasm.go`)
   - High-performance WebAssembly execution engine
   - Native WASI (WebAssembly System Interface) support
   - Socket operations and network I/O capabilities
   - Advanced type conversion system for Go interoperability

4. **TEE Integration** (`dtvm/tee_sev_snp.go`)
   - AMD SEV-SNP attestation generation
   - Secure quote provider integration
   - Cryptographic proof of execution integrity

5. **Protocol Buffers** (`proto/wasm/wasm.proto`)
   - Defines gRPC service interface
   - Message types for execution requests/responses
   - Type-safe value representations with complex data types

### WasmEdge Runtime Features

- **High Performance**: Optimized WebAssembly runtime for server-side applications
- **WASI Compliance**: Full WebAssembly System Interface support for:
  - File system operations
  - Environment variable access
  - Command-line argument processing
  - Socket and network operations
  - Clock and random number generation
- **Advanced Type System**: Support for complex data types including:
  - Primitive types (u8, i32, i64, f32, f64)
  - Strings and byte arrays
  - Vectors and complex structures
  - JSON serialization for complex return values
- **Host Functions**: Extensible host function interface for:
  - HTTP requests and web API access
  - Memory management operations
  - Custom system integrations

### Security Features

- **Trusted Execution Environment**: Runs within AMD SEV-SNP secure enclaves
- **Cryptographic Attestation**: Generates verifiable proofs of execution
- **Input/Output Integrity**: SHA-256 hashing of all inputs and outputs
- **Deterministic Execution**: Consistent results across multiple runs
- **Sandboxed Execution**: WasmEdge provides secure isolation for WASM modules
- **WASI Security**: Controlled system access through WASI capabilities

## Prerequisites

- Go 1.21 or later
- Rust toolchain with `wasm32-wasip1` target
- WasmEdge runtime (automatically installed via CI)
- AMD SEV-SNP capable hardware (for production)
- Protocol Buffers compiler (`protoc`)
- Buf CLI tool for proto management

## Installation

1. **Clone the repository**:

```bash
git clone https://github.com/IntelliXLabs/wasmvm-tee.git
cd wasmvm-tee
```

2. **Follow the Development section** for complete setup including:
   - Go installation
   - Rust and WASM target setup
   - WasmEdge runtime installation
   - Project dependencies

3. **Build and run**:

```bash
make build
./bin/sev_snp_server
```

## Usage

### Starting the Server

#### Using the compiled binary (recommended)

```bash
# Build first
make build

# Start with default port (50051)
./bin/sev_snp_server

# Start with custom port
./bin/sev_snp_server -port 8080
```

## Development

### Prerequisites Installation

1. **Install Go** (1.21 or later):

```bash
# Download and install Go from https://golang.org/dl/
# Or use package manager:
# macOS: brew install go
# Ubuntu: sudo apt install golang-go
go version  # Verify installation
```

2. **Install Rust and WASM target**:

```bash
# Install Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source ~/.cargo/env

# Add WASM target
rustup target add wasm32-wasip1
```

3. **Install WasmEdge runtime**:

```bash
# Install WasmEdge using official installer
curl -sSf https://raw.githubusercontent.com/WasmEdge/WasmEdge/master/utils/install.sh | bash -s -- -v 0.14.0

# Make the installed binary available in current session
source $HOME/.wasmedge/env
```

4. **Install project dependencies**:

```bash
make deps
```

### Build and Test

1. **Build the project**:

```bash
make build
```

2. **Run tests**:

```bash
make test
```

### Additional Commands

- **Generate Protocol Buffers**: `make proto`
- **Clean build artifacts**: `make clean`
- **Format code**: `make proto-format`
- **Lint code**: `make proto-lint`

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make proto-lint`
6. Submit a pull request

## License

This project is licensed under the terms specified in the repository.

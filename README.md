# WASMVM-TEE: Trusted Execution Environment for WASMVM

A secure gRPC service that executes WASMVM (WebAssembly Virtual Machine) bytecode within a Trusted Execution Environment (TEE) using AMD SEV-SNP technology. This project provides cryptographic attestation of execution results, ensuring integrity and confidentiality of computations.

## Architecture

### Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   gRPC Client   │───▶│  WASMVM-TEE      │───▶│   AMD SEV-SNP   │
│                 │    │  Server         │    │   Attestation   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   WASMVM Runtime  │
                       │   Execution     │
                       └─────────────────┘
```

### Detailed Flow

```
┌─────────────────┐
│   gRPC Client   │
│                 │
└─────────┬───────┘
          │ WASMVMExecutionRequest
          ▼
┌─────────────────┐
│  WASMVM-TEE      │
│  Server         │
│  ┌─────────────┐│
│  │ Validation  ││
│  └─────────────┘│
└─────────┬───────┘
          │ Bytecode + Inputs
          ▼
┌─────────────────┐
│   WASMVM Runtime  │
│   Execution     │
│  ┌─────────────┐│
│  │ WASM Engine ││
│  └─────────────┘│
└─────────┬───────┘
          │ Results
          ▼
┌─────────────────┐
│   AMD SEV-SNP   │
│   Attestation   │
│  ┌─────────────┐│
│  │ Quote Gen   ││
│  └─────────────┘│
└─────────┬───────┘
          │ WASMVMExecutionResponse
          ▼
┌─────────────────┐
│   gRPC Client   │
│   (Response)    │
└─────────────────┘
```

### Components

1. **gRPC Server** (`cmd/sev_snp_server/main.go`)
   - Listens for WASMVM execution requests
   - Configurable port via command line arguments
   - Supports gRPC reflection for debugging

2. **WASMVM TEE Service** (`dtvm/server.go`)
   - Implements `WASMVMTeeService` gRPC interface
   - Handles bytecode execution in secure environment
   - Generates cryptographic attestations

3. **TEE Integration** (`dtvm/tee_sev_snp.go`)
   - AMD SEV-SNP attestation generation
   - Secure quote provider integration
   - Cryptographic proof of execution integrity

4. **Protocol Buffers** (`proto/dtvm/dtvm.proto`)
   - Defines gRPC service interface
   - Message types for execution requests/responses
   - Type-safe value representations

### Security Features

- **Trusted Execution Environment**: Runs within AMD SEV-SNP secure enclaves
- **Cryptographic Attestation**: Generates verifiable proofs of execution
- **Input/Output Integrity**: SHA-256 hashing of all inputs and outputs
- **Deterministic Execution**: Consistent results across multiple runs
- **Gas Limiting**: Optional resource consumption controls

## Prerequisites

- Go 1.23.2 or later
- AMD SEV-SNP capable hardware (for production)
- Protocol Buffers compiler (`protoc`)
- Buf CLI tool for proto management

## Installation

1. Clone the repository:

```bash
git clone https://github.com/IntelliXLabs/wasmvm-tee.git
cd wasmvm-tee
```

2. Install dependencies:

```bash
make deps
```

3. Generate protobuf files:

```bash
make proto
```

## Building

### Build the server

```bash
make build
```

This will compile the SEV-SNP server from `cmd/sev_snp_server/main.go` and create the binary at `bin/sev_snp_server`.

### Or build manually

```bash
go build -o bin/sev_snp_server cmd/sev_snp_server/main.go
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

### Example Client Request

The server expects `WASMVMExecutionRequest` messages with the following structure:

```protobuf
message WASMVMExecutionRequest {
  WASMVMExecution execution = 1;
  WASMVMRuntimeConfig runtime_config = 2;
}
```

Where:

- `execution.bytecode`: Base64-encoded WASMVM bytecode
- `execution.fn_name`: Function name to execute
- `execution.inputs`: Base64-encoded input parameters
- `runtime_config.mode`: Execution mode (interpreter/compiler)
- `runtime_config.gas_limit`: Optional gas consumption limits

### Response Format

The server returns `WASMVMExecutionResponse` containing:

- `request_id`: Request tracking identifier
- `result.output_values`: Typed execution results
- `result.attestation`: SEV-SNP attestation report (JSON)
- `result.report_data`: Cryptographic hash of inputs+outputs

## Development

### Generate Protocol Buffers

```bash
make proto
```

### Format Proto Files

```bash
make proto-format
```

### Lint Proto Files

```bash
make proto-lint
```

### Run Tests

```bash
make test
```

### Clean Generated Files

```bash
make clean
```

## API Reference

### gRPC Service

```protobuf
service WASMVMTeeService {
  rpc Execute(WASMVMExecutionRequest) returns (WASMVMExecutionResponse);
}
```

### Supported Value Types

- `VALUE_TYPE_INT32`: 32-bit signed integer
- `VALUE_TYPE_INT64`: 64-bit signed integer  
- `VALUE_TYPE_FLOAT32`: 32-bit floating point
- `VALUE_TYPE_FLOAT64`: 64-bit floating point

### Execution Modes

- `WASMVM_MODE_INTERP_UNSPECIFIED`: Interpreter mode (default)
- `WASMVM_MODE_SINGLEPASS`: Single-pass compilation
- `WASMVM_MODE_MULTIPASS`: Multi-pass compilation

## Security Considerations

1. **Hardware Requirements**: Production deployments require AMD SEV-SNP capable hardware
2. **Attestation Verification**: Clients should verify attestation reports against known measurements
3. **Input Validation**: All inputs are validated and sanitized before execution
4. **Resource Limits**: Configure appropriate gas limits to prevent resource exhaustion

## Dependencies

- [WASMVM](https://github.com/WASMVMStack/WASMVM): WebAssembly Virtual Machine with SmartCogent - The core WebAssembly runtime engine
- [wasmvm-go](https://github.com/IntelliXLabs/wasmvm-go): WASMVM runtime library
- [go-sev-guest](https://github.com/google/go-sev-guest): AMD SEV-SNP attestation
- [gRPC](https://grpc.io/): High-performance RPC framework
- [Protocol Buffers](https://protobuf.dev/): Serialization framework

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run `make test` and `make proto-lint`
6. Submit a pull request

## License

This project is licensed under the terms specified in the repository.

## Support

For issues and questions:

- Create an issue on GitHub
- Check existing documentation
- Review the protocol buffer definitions in `proto/dtvm/dtvm.proto`

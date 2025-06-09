# WASM Types

This directory contains the generated protobuf files for the WASMVM TEE service.

## Generated Files

- `wasm_server.pb.go` - Core protobuf message definitions
- `wasm_server_grpc.pb.go` - gRPC service definitions  
- `wasm_server.pb.gw.go` - gRPC-Gateway HTTP handlers
- `wasm_server.swagger.json` - OpenAPI/Swagger documentation for server endpoints
- `wasm_input.pb.go` - Input type definitions for WASM values
- `wasm_input.swagger.json` - OpenAPI/Swagger documentation for input types

## Regenerating Files

To regenerate these files after modifying the proto definitions:

```bash
make clean  # Clean existing generated files
make proto  # Generate new files
```

## Package Structure

All generated files use the `types` package and are imported as:

```go
import "github.com/IntelliXLabs/wasmvm-tee/wasm/types"
```

## Note

These files are automatically generated from the proto definitions in `proto/wasm/`.
Do not edit these files directly - modify the source `.proto` files instead.

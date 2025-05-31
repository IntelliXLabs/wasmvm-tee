package wasmedge

import (
	"os"
	"testing"
)

func TestExecuteWasmRunFunction(t *testing.T) {
	// Load rust_host_func.wasm file from the correct build path
	wasmBytes, err := os.ReadFile("../wasmedge/rust_host_func/target/wasm32-wasip1/release/rust_host_func.wasm")
	if err != nil {
		t.Fatalf("Failed to read WASM file: %v", err)
	}

	// Execute run function with empty parameters
	result, err := ExecuteWasm(wasmBytes, "run", []any{})
	if err != nil {
		t.Fatalf("Failed to execute run function: %v", err)
	}

	t.Logf("Successfully executed run function. Result: %v", result)
}

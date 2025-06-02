package wasm

import (
	"fmt"
	"os"
	"testing"
)

// TestHttpFunctions tests the newly added HTTP functions
func TestHttpFunctions(t *testing.T) {
	// Read the compiled WASM file
	wasmBytes, err := os.ReadFile("rust_host_func/target/wasm32-wasip1/release/rust_host_func.wasm")
	if err != nil {
		t.Fatalf("Failed to read WASM file: %v", err)
	}

	fmt.Printf("Loaded WASM file, size: %d bytes\n", len(wasmBytes))

	// Test HTTP GET request
	t.Run("test_http_get", func(t *testing.T) {
		fmt.Println("\n=== Testing HTTP GET Request ===")

		results, err := ExecuteWasm(wasmBytes, "test_http_get", []any{})
		if err != nil {
			t.Errorf("Failed to execute test_http_get: %v", err)
			return
		}

		if len(results) > 0 {
			if response, ok := results[0].(string); ok {
				fmt.Printf("HTTP GET Response:\n%s\n", response)
			} else {
				fmt.Printf("HTTP GET Response (type %T): %v\n", results[0], results[0])
			}
		} else {
			t.Error("No results returned from test_http_get")
		}
	})

	// Test HTTP POST request
	t.Run("test_http_post", func(t *testing.T) {
		fmt.Println("\n=== Testing HTTP POST Request ===")

		results, err := ExecuteWasm(wasmBytes, "test_http_post", []any{})
		if err != nil {
			t.Errorf("Failed to execute test_http_post: %v", err)
			return
		}

		if len(results) > 0 {
			if response, ok := results[0].(string); ok {
				fmt.Printf("HTTP POST Response:\n%s\n", response)
			} else {
				fmt.Printf("HTTP POST Response (type %T): %v\n", results[0], results[0])
			}
		} else {
			t.Error("No results returned from test_http_post")
		}
	})

	// Test HTTP request with custom headers
	t.Run("test_http_with_headers", func(t *testing.T) {
		fmt.Println("\n=== Testing HTTP Request with Custom Headers ===")

		results, err := ExecuteWasm(wasmBytes, "test_http_with_headers", []any{})
		if err != nil {
			t.Errorf("Failed to execute test_http_with_headers: %v", err)
			return
		}

		if len(results) > 0 {
			if response, ok := results[0].(string); ok {
				fmt.Printf("HTTP Headers Response:\n%s\n", response)
			} else {
				fmt.Printf("HTTP Headers Response (type %T): %v\n", results[0], results[0])
			}
		} else {
			t.Error("No results returned from test_http_with_headers")
		}
	})

	// Test the original call_google function for comparison
	t.Run("call_google_comparison", func(t *testing.T) {
		fmt.Println("\n=== Testing Original call_google Function (for comparison) ===")

		results, err := ExecuteWasm(wasmBytes, "call_google", []any{})
		if err != nil {
			t.Errorf("Failed to execute call_google: %v", err)
			return
		}

		if len(results) > 0 {
			fmt.Printf("call_google result: %v\n", results[0])
		} else {
			t.Error("No results returned from call_google")
		}
	})
}

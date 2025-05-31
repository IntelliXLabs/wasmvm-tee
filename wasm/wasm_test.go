package wasm

import (
	"encoding/json"
	"os"
	reflect "reflect"
	"testing"
)

var wasmFilePath = "../wasm/rust_host_func/target/wasm32-wasip1/release/rust_host_func.wasm"

// TestExecuteWasmAllFunctions - Comprehensive test for all WASM functions
func TestExecuteWasmAllFunctions(t *testing.T) {
	// Check if WASM file exists
	if _, err := os.Stat(wasmFilePath); os.IsNotExist(err) {
		t.Fatalf("WASM file not found at %s. Please ensure it is compiled and the path is correct.", wasmFilePath)
	}

	// Load WASM file
	wasmBytes, err := os.ReadFile(wasmFilePath)
	if err != nil {
		t.Fatalf("Failed to read WASM file %s: %v", wasmFilePath, err)
	}

	// Test 1: call_google function
	t.Run("call_google", func(t *testing.T) {
		results, err := ExecuteWasm(wasmBytes, "call_google", []any{})
		if err != nil {
			t.Fatalf("Failed to execute 'call_google' function: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result from 'call_google' function, got %d", len(results))
		}

		if results[0].(int32) != 111 {
			t.Fatalf("Expected 111, got %d", results[0].(int32))
		}

		t.Logf("✓ call_google executed successfully. Result: %d", results[0].(int32))
	})

	// Test 2: say function
	t.Run("say", func(t *testing.T) {
		inputName := "WasmEdge"
		expectedOutput := "hello " + inputName
		params := []any{inputName}

		results, err := ExecuteWasm(wasmBytes, "say", params)
		if err != nil {
			t.Fatalf("Failed to execute 'say' function: %v", err)
		}

		if len(results) != 1 {
			t.Fatalf("Expected 1 result from 'say' function, got %d", len(results))
		}

		resultStr, ok := results[0].(string)
		if !ok {
			t.Fatalf("Expected result to be a string, got %T", results[0])
		}

		if resultStr != expectedOutput {
			t.Errorf("Unexpected output from 'say' function. Expected '%s', got '%s'", expectedOutput, resultStr)
		}

		t.Logf("✓ say function executed successfully. Input: '%s', Output: '%s'", inputName, resultStr)
	})

	// Test 3: JSON-based complex types function
	t.Run("process_complex_types_json", func(t *testing.T) {
		// Prepare input parameters
		inputU8 := uint8(5)
		inputBytes := []byte{10, 20, 30}
		inputString := "TestStr"
		inputVector := []int32{1, 2, 3, 4, 5}

		params := []any{inputU8, inputBytes, inputString, inputVector}

		// Execute the JSON-returning function
		results, err := ExecuteWasm(wasmBytes, "process_complex_types_json", params)
		if err != nil {
			t.Fatalf("Failed to execute 'process_complex_types_json' function: %v", err)
		}

		// Expecting 1 result (JSON string)
		if len(results) != 1 {
			t.Fatalf("Expected 1 result from 'process_complex_types_json' function, got %d. Results: %v", len(results), results)
		}

		// Assert the result is a string
		jsonResult, ok := results[0].(string)
		if !ok {
			t.Fatalf("Expected result to be string, got %T", results[0])
		}

		// Parse and validate the JSON result
		var parsedResult map[string]any
		err = json.Unmarshal([]byte(jsonResult), &parsedResult)
		if err != nil {
			t.Fatalf("Failed to parse JSON result: %v. JSON: %s", err, jsonResult)
		}

		// Define expected values
		expectedProcessedU8 := float64(5 + 10) // JSON numbers are float64
		expectedProcessedBytesLen := float64(len(inputBytes))
		expectedAppendedString := inputString + " processed"
		var expectedSumOfVector float64
		for _, v := range inputVector {
			expectedSumOfVector += float64(v)
		}
		runes := []rune(inputString)
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		expectedOriginalStringReversed := string(runes)

		expectedReturnedVector := make([]any, len(inputVector))
		for i, v := range inputVector {
			expectedReturnedVector[i] = float64(v * 2) // JSON arrays contain float64
		}

		// Validate each field in the JSON result
		if processedU8, ok := parsedResult["processed_u8"].(float64); !ok || processedU8 != expectedProcessedU8 {
			t.Errorf("Unexpected processed_u8. Expected %v, got %v", expectedProcessedU8, parsedResult["processed_u8"])
		}

		if bytesLen, ok := parsedResult["bytes_len"].(float64); !ok || bytesLen != expectedProcessedBytesLen {
			t.Errorf("Unexpected bytes_len. Expected %v, got %v", expectedProcessedBytesLen, parsedResult["bytes_len"])
		}

		if appendedString, ok := parsedResult["appended_string"].(string); !ok || appendedString != expectedAppendedString {
			t.Errorf("Unexpected appended_string. Expected '%s', got '%v'", expectedAppendedString, parsedResult["appended_string"])
		}

		if vectorSum, ok := parsedResult["vector_sum"].(float64); !ok || vectorSum != expectedSumOfVector {
			t.Errorf("Unexpected vector_sum. Expected %v, got %v", expectedSumOfVector, parsedResult["vector_sum"])
		}

		if reversedString, ok := parsedResult["reversed_string"].(string); !ok || reversedString != expectedOriginalStringReversed {
			t.Errorf("Unexpected reversed_string. Expected '%s', got '%v'", expectedOriginalStringReversed, parsedResult["reversed_string"])
		}

		if doubledVector, ok := parsedResult["doubled_vector"].([]any); !ok || !reflect.DeepEqual(doubledVector, expectedReturnedVector) {
			t.Errorf("Unexpected doubled_vector. Expected %v, got %v", expectedReturnedVector, parsedResult["doubled_vector"])
		}

		t.Logf("✓ process_complex_types_json executed successfully.")
		t.Logf("  JSON Result: %s", jsonResult)
	})

	t.Logf("🎉 All WASM function tests completed successfully!")
}

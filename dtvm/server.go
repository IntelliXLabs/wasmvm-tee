package dtvm

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/IntelliXLabs/dtvm-go"
)

var _ DTVMTeeServiceServer = (*Server)(nil)

type Server struct {
	UnimplementedDTVMTeeServiceServer
}

// Execute handles DTVM execution requests in TEE environment
// Validates request, decodes bytecode and inputs, executes DTVM, and returns results with attestation
func (s *Server) Execute(ctx context.Context, req *DTVMExecutionRequest) (*DTVMExecutionResponse, error) {
	// Validate request
	if req.Execution == nil {
		return nil, fmt.Errorf("execution request is nil")
	}
	if req.RuntimeConfig == nil {
		return nil, fmt.Errorf("runtime config is nil")
	}

	// Decode bytecode
	bytecode, err := base64.StdEncoding.DecodeString(req.Execution.Bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode: %v", err)
	}

	// Execute DTVM (pass the entire execution object)
	result, err := s.executeDTVM(req.RuntimeConfig, bytecode, req.Execution.FnName, req.Execution)
	if err != nil {
		return nil, fmt.Errorf("failed to execute DTVM: %v", err)
	}

	// Build response
	response := &DTVMExecutionResponse{
		RequestId: req.Execution.RequestId,
		Result:    result,
	}

	return response, nil
}

// executeDTVM performs the actual DTVM execution with the specified configuration
// Creates runtime, loads module from bytecode buffer, and executes the specified function
func (s *Server) executeDTVM(config *DTVMRuntimeConfig, bytecode []byte, fnName string, execution *DTVMExecution) (*DTVMExecutionResult, error) {
	// Create runtime configuration
	runtimeConfig := dtvm.NewRuntimeConfig(int32(config.Mode))
	defer runtimeConfig.Delete()

	// Create runtime
	runtime := dtvm.NewRuntime(runtimeConfig)
	defer runtime.Delete()

	// Load module directly from buffer (no temporary file needed)
	moduleName := fmt.Sprintf("dtvm_module_%d", time.Now().UnixNano())
	module, err := runtime.LoadModuleFromBuffer(moduleName, bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to load module from buffer: %v", err)
	}
	defer module.Delete()

	// Create isolation environment
	isolation := runtime.CreateIsolation()
	defer isolation.Delete()

	// Create instance with or without gas limit based on configuration
	var instance *dtvm.Instance
	if config.GasLimit.UseGasLimit {
		instance, err = isolation.CreateInstanceWithGas(module, uint64(config.GasLimit.Limit))
	} else {
		instance, err = isolation.CreateInstance(module)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %v", err)
	}
	defer instance.Delete()

	// Execute the specified WASM function
	results, err := instance.CallWasmFuncByName(runtime, fnName, execution.Inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to call function %s: %v", fnName, err)
	}

	// Convert DTVM results to protobuf format
	outputValues := make([]*Value, len(results))
	for i, result := range results {
		outputValues[i] = s.convertDTVMValue(&result)
	}

	// Calculate cryptographic hashes for integrity verification
	inputHash, err := s.calculateStandardHash(execution)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate input hash: %v", err)
	}

	outputHash, err := s.calculateOutputHash(outputValues)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate output hash: %v", err)
	}

	var combined [64]byte
	copy(combined[:32], inputHash[:])
	copy(combined[32:], outputHash[:])

	attestation, err := generateAttestation(combined)
	if err != nil {
		return nil, fmt.Errorf("failed to generate attestation: %v", err)
	}

	// Build execution result with attestation data
	executionResult := &DTVMExecutionResult{
		Inputs:       encodeStringsToBase64(execution.Inputs),
		OutputValues: outputValues,
		Attestation:  string(attestation),
		ReportData:   hex.EncodeToString(combined[:]),
	}

	return executionResult, nil
}

// convertDTVMValue converts DTVM native value types to protobuf Value format
// Handles type detection and proper value extraction for different data types
func (s *Server) convertDTVMValue(value *dtvm.Value) *Value {
	// Convert DTVM Value type to protobuf Value
	// This needs to be adjusted based on the actual structure of dtvm.Value
	switch v := value.Value.(type) {
	case int32:
		return &Value{
			Type:  ValueType_VALUE_TYPE_INT32,
			Value: &Value_Int32Value{Int32Value: v},
		}
	case int64:
		return &Value{
			Type:  ValueType_VALUE_TYPE_INT64,
			Value: &Value_Int64Value{Int64Value: v},
		}
	case float32:
		return &Value{
			Type:  ValueType_VALUE_TYPE_FLOAT32,
			Value: &Value_Float32Value{Float32Value: v},
		}
	case float64:
		return &Value{
			Type:  ValueType_VALUE_TYPE_FLOAT64,
			Value: &Value_Float64Value{Float64Value: v},
		}
	default:
		// Default to int32 type
		log.Printf("Unknown value type, defaulting to int32: %T", v)
		return &Value{
			Type:  ValueType_VALUE_TYPE_INT32,
			Value: &Value_Int32Value{Int32Value: 0},
		}
	}
}

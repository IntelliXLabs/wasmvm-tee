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

	// Execute DTVM (pass the entire execution object)
	result, err := s.executeDTVM(req.RuntimeConfig, req.Execution)
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
func (s *Server) executeDTVM(config *DTVMRuntimeConfig, execution *DTVMExecution) (*DTVMExecutionResult, error) {
	// Decode bytecode
	bytecode, err := base64.StdEncoding.DecodeString(execution.Bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode: %v", err)
	}

	// Execute WASM function and get results
	outputValues, err := s.executeWASMFunction(config, bytecode, execution.FnName, execution.Inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WASM function: %v", err)
	}

	// Generate attestation based on execution data
	attestation, reportData, err := s.buildAttestationByExecution(execution, outputValues)
	if err != nil {
		return nil, fmt.Errorf("failed to build attestation: %v", err)
	}

	return &DTVMExecutionResult{
		Inputs:       execution.Inputs,
		OutputValues: outputValues,
		Attestation:  attestation,
		ReportData:   reportData,
	}, nil
}

// executeWASMFunction executes WASM bytecode in DTVM runtime and returns converted results
func (s *Server) executeWASMFunction(config *DTVMRuntimeConfig, bytecode []byte, fnName string, inputs []string) ([]*Value, error) {
	// Create runtime configuration
	runtimeConfig := dtvm.NewRuntimeConfig(int32(config.Mode))
	defer runtimeConfig.Delete()

	// Create runtime
	runtime := dtvm.NewRuntime(runtimeConfig)
	defer runtime.Delete()

	// Load module from bytecode buffer
	moduleName := fmt.Sprintf("dtvm_module_%d", time.Now().UnixNano())
	module, err := runtime.LoadModuleFromBuffer(moduleName, bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to load module from buffer: %v", err)
	}

	defer module.Delete()

	// Create instance with isolation
	instance, err := s.createInstance(runtime, module, config)
	if err != nil {
		return nil, err
	}
	defer instance.Delete()

	// Execute the WASM function
	results, err := instance.CallWasmFuncByName(runtime, fnName, inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to call function %s: %v", fnName, err)
	}

	// Convert DTVM results to protobuf format
	return s.convertResults(results), nil
}

// createInstance creates DTVM instance with isolation and optional gas limit
func (s *Server) createInstance(runtime *dtvm.Runtime, module *dtvm.Module, config *DTVMRuntimeConfig) (*dtvm.Instance, error) {
	// Create isolation environment
	isolation := runtime.CreateIsolation()
	defer isolation.Delete()

	// Create instance with or without gas limit
	var instance *dtvm.Instance
	var err error

	if config.GasLimit.UseGasLimit {
		instance, err = isolation.CreateInstanceWithGas(module, uint64(config.GasLimit.Limit))
	} else {
		instance, err = isolation.CreateInstance(module)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %v", err)
	}

	return instance, nil
}

// convertResults converts DTVM results to protobuf format
func (s *Server) convertResults(results []dtvm.Value) []*Value {
	outputValues := make([]*Value, len(results))
	for i, result := range results {
		outputValues[i] = s.convertDTVMValue(&result)
	}

	return outputValues
}

// buildAttestationByExecution creates attestation data based on execution inputs and outputs
// Calculates cryptographic hashes for integrity verification and generates TEE attestation
func (s *Server) buildAttestationByExecution(execution *DTVMExecution, outputValues []*Value) (string, string, error) {
	// Calculate cryptographic hashes for integrity verification
	inputHash, err := s.calculateStandardHash(execution)
	if err != nil {
		return "", "", fmt.Errorf("failed to calculate input hash: %v", err)
	}

	outputHash, err := s.calculateOutputHash(outputValues)
	if err != nil {
		return "", "", fmt.Errorf("failed to calculate output hash: %v", err)
	}

	// Combine input and output hashes
	combined := s.combineHashes(inputHash, outputHash)

	// Generate TEE attestation
	attestation, err := generateAttestation(combined)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate attestation: %v", err)
	}

	return string(attestation), hex.EncodeToString(combined[:]), nil
}

// combineHashes combines input and output hashes into a single 64-byte array
func (s *Server) combineHashes(inputHash, outputHash [32]byte) [64]byte {
	var combined [64]byte
	copy(combined[:32], inputHash[:])
	copy(combined[32:], outputHash[:])
	return combined
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

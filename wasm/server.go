package wasm

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/IntelliXLabs/wasmvm-tee/wasm/types"
)

var _ types.WASMVMTeeServiceServer = (*Server)(nil)

type Server struct {
	types.UnimplementedWASMVMTeeServiceServer
}

// Execute handles WASMVM execution requests in TEE environment
// Validates request, decodes bytecode and inputs, executes WASMVM, and returns results with attestation
func (s *Server) Execute(ctx context.Context, req *types.WASMVMExecutionRequest) (*types.WASMVMExecutionResponse, error) {
	// Validate request
	if req.Execution == nil {
		return nil, fmt.Errorf("execution request is nil")
	}

	// Execute WASMVM (pass the entire execution object)
	result, err := s.executeWASMVM(req.Execution)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WASMVM: %v", err)
	}

	// Build response
	response := &types.WASMVMExecutionResponse{
		RequestId: req.Execution.RequestId,
		Result:    result,
	}

	return response, nil
}

// executeWASMVM performs the actual WASMVM execution with WasmEdge
// Decodes bytecode, converts inputs, and executes the specified function
func (s *Server) executeWASMVM(execution *types.WASMVMExecution) (*types.WASMVMExecutionResult, error) {
	// Decode bytecode
	bytecode, err := base64.StdEncoding.DecodeString(execution.Bytecode)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bytecode: %v", err)
	}

	// Convert string inputs to appropriate types for WasmEdge
	params, err := ConvertWasmValuesToInterface(execution.Inputs)
	if err != nil {
		return nil, fmt.Errorf("failed to convert inputs: %v", err)
	}

	// Execute WASM function using WasmEdge and get proto Value results
	output, err := ExecuteWasm(bytecode, execution.FnName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to execute WASM function: %v", err)
	}

	outputValues, err := ConvertBindgenExecuteResultToWasmValues(output)
	if err != nil {
		return nil, fmt.Errorf("failed to convert output values: %v", err)
	}

	// Generate attestation based on execution data
	attestation, reportData, err := s.buildAttestationByExecution(execution, outputValues)
	if err != nil {
		return nil, fmt.Errorf("failed to build attestation: %v", err)
	}

	return &types.WASMVMExecutionResult{
		Inputs:       execution.Inputs,
		OutputValues: outputValues,
		Attestation:  attestation,
		ReportData:   reportData,
	}, nil
}

// buildAttestationByExecution creates attestation data based on execution inputs and outputs
// Calculates cryptographic hashes for integrity verification and generates TEE attestation
func (s *Server) buildAttestationByExecution(execution *types.WASMVMExecution, outputValues []*types.WasmValue) (string, string, error) {
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

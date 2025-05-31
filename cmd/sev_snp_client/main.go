package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/IntelliXLabs/wasmvm-tee/dtvm"
)

// Config represents the complete client configuration matching gRPC request structure
type Config struct {
	Server        ServerConfig        `json:"server"`
	Execution     DTVMExecutionConfig `json:"execution"`
	RuntimeConfig DTVMRuntimeConfig   `json:"runtime_config"`
}

// ServerConfig contains server connection settings
type ServerConfig struct {
	Address string `json:"address"`
	Timeout int    `json:"timeout_seconds"`
}

// DTVMExecutionConfig maps directly to DTVMExecution protobuf message
type DTVMExecutionConfig struct {
	Version      string   `json:"version"`
	RequestId    string   `json:"request_id"`
	Bytecode     string   `json:"bytecode"`
	BytecodeFile string   `json:"bytecode_file,omitempty"`
	FnName       string   `json:"fn_name"`
	Inputs       []string `json:"inputs"`
	Timestamp    int64    `json:"timestamp"`
}

// DTVMRuntimeConfig maps directly to DTVMRuntimeConfig protobuf message
type DTVMRuntimeConfig struct {
	Mode     string         `json:"mode"`
	GasLimit GasLimitConfig `json:"gas_limit"`
}

// GasLimitConfig maps directly to GasLimitConfig protobuf message
type GasLimitConfig struct {
	UseGasLimit bool  `json:"use_gas_limit"`
	Limit       int64 `json:"limit"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address: "localhost:50051",
			Timeout: 30,
		},
		Execution: DTVMExecutionConfig{
			Version:   "1.0",
			RequestId: "",
			Bytecode:  "",
			FnName:    "add",
			Inputs:    []string{"5", "3"},
			Timestamp: 0,
		},
		RuntimeConfig: DTVMRuntimeConfig{
			Mode: "interpreter",
			GasLimit: GasLimitConfig{
				UseGasLimit: false,
				Limit:       0,
			},
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file not found, using default configuration\n")
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// Validate and set defaults
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %v", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Server validation
	if c.Server.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}
	if c.Server.Timeout <= 0 {
		c.Server.Timeout = 30 // default timeout
	}

	// Execution validation
	if c.Execution.Version == "" {
		c.Execution.Version = "1.0"
	}
	if c.Execution.FnName == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	if len(c.Execution.Inputs) == 0 {
		return fmt.Errorf("at least one input is required")
	}

	// Generate request ID if not provided
	if c.Execution.RequestId == "" {
		c.Execution.RequestId = generateRequestID()
	}

	// Set timestamp if not provided
	if c.Execution.Timestamp == 0 {
		c.Execution.Timestamp = time.Now().Unix()
	}

	// Runtime config validation
	switch c.RuntimeConfig.Mode {
	case "interpreter", "singlepass", "multipass":
		// valid modes
	case "":
		c.RuntimeConfig.Mode = "interpreter" // default
	default:
		return fmt.Errorf("invalid runtime mode: %s", c.RuntimeConfig.Mode)
	}

	if c.RuntimeConfig.GasLimit.UseGasLimit && c.RuntimeConfig.GasLimit.Limit <= 0 {
		return fmt.Errorf("gas limit must be positive when enabled")
	}

	return nil
}

// ToExecutionMode converts string mode to DTVMMode enum
func (c *Config) ToExecutionMode() dtvm.DTVMMode {
	switch c.RuntimeConfig.Mode {
	case "singlepass":
		return dtvm.DTVMMode_DTVM_MODE_SINGLEPASS
	case "multipass":
		return dtvm.DTVMMode_DTVM_MODE_MULTIPASS
	default:
		return dtvm.DTVMMode_DTVM_MODE_INTERP_UNSPECIFIED
	}
}

// LoadBytecode loads bytecode from file or returns embedded bytecode
func (c *Config) LoadBytecode() (string, error) {
	// If bytecode file is specified, load from file
	if c.Execution.BytecodeFile != "" {
		data, err := os.ReadFile(c.Execution.BytecodeFile)
		if err != nil {
			return "", fmt.Errorf("failed to read bytecode file: %v", err)
		}
		return string(data), nil
	}

	// If bytecode is specified directly, use it
	if c.Execution.Bytecode != "" {
		return c.Execution.Bytecode, nil
	}

	// Otherwise, use default sample bytecode
	return createSampleBytecode(), nil
}

// ToGRPCRequest converts config to gRPC request
func (c *Config) ToGRPCRequest() (*dtvm.DTVMExecutionRequest, error) {
	// Load bytecode
	bytecode, err := c.LoadBytecode()
	if err != nil {
		return nil, fmt.Errorf("failed to load bytecode: %v", err)
	}

	// Build gRPC request
	request := &dtvm.DTVMExecutionRequest{
		Execution: &dtvm.DTVMExecution{
			Version:   c.Execution.Version,
			RequestId: c.Execution.RequestId,
			Bytecode:  bytecode,
			FnName:    c.Execution.FnName,
			Inputs:    c.Execution.Inputs,
			Timestamp: c.Execution.Timestamp,
		},
		RuntimeConfig: &dtvm.DTVMRuntimeConfig{
			Mode: c.ToExecutionMode(),
			GasLimit: &dtvm.GasLimitConfig{
				UseGasLimit: c.RuntimeConfig.GasLimit.UseGasLimit,
				Limit:       c.RuntimeConfig.GasLimit.Limit,
			},
		},
	}

	return request, nil
}

func main() {
	var (
		configPath     = flag.String("config", "config/client.json", "Path to configuration file")
		generateConfig = flag.Bool("generate-config", false, "Generate default configuration file and exit")
		listConfigs    = flag.Bool("list-configs", false, "List available configuration files")
		validateConfig = flag.Bool("validate", false, "Validate configuration file and exit")
		showRequest    = flag.Bool("show-request", false, "Show the gRPC request that would be sent")
	)
	flag.Parse()

	// List available configs if requested
	if *listConfigs {
		listAvailableConfigs()
		return
	}

	// Generate default config if requested
	if *generateConfig {
		if err := generateDefaultConfig(*configPath); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		fmt.Printf("Default configuration generated at: %s\n", *configPath)
		return
	}

	// Load configuration
	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate config if requested
	if *validateConfig {
		fmt.Printf("Configuration is valid ✅\n")
		displayConfiguration(config, *configPath)
		return
	}

	// Show request if requested
	if *showRequest {
		request, err := config.ToGRPCRequest()
		if err != nil {
			log.Fatalf("Failed to build request: %v", err)
		}
		displayGRPCRequest(request)
		return
	}

	// Display configuration
	displayConfiguration(config, *configPath)

	// Execute request
	if err := executeRequest(config); err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
}

// generateDefaultConfig generates a default configuration file
func generateDefaultConfig(configPath string) error {
	config := DefaultConfig()
	return SaveConfig(config, configPath)
}

// listAvailableConfigs lists available configuration files
func listAvailableConfigs() {
	fmt.Printf("Available Configuration Files:\n")
	fmt.Printf("==============================\n")

	configDir := "config"
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		fmt.Printf("No config directory found. Run with -generate-config to create default configs.\n")
		return
	}

	files, err := os.ReadDir(configDir)
	if err != nil {
		fmt.Printf("Error reading config directory: %v\n", err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			configPath := filepath.Join(configDir, file.Name())
			config, err := LoadConfig(configPath)
			if err != nil {
				fmt.Printf("❌ %s (invalid: %v)\n", file.Name(), err)
				continue
			}

			fmt.Printf("✅ %s\n", file.Name())
			fmt.Printf("   Server: %s\n", config.Server.Address)
			fmt.Printf("   Function: %s\n", config.Execution.FnName)
			fmt.Printf("   Mode: %s\n", config.RuntimeConfig.Mode)
			fmt.Printf("   Inputs: %v\n", config.Execution.Inputs)
			fmt.Printf("\n")
		}
	}
}

// displayConfiguration displays the current configuration
func displayConfiguration(config *Config, configPath string) {
	fmt.Printf("DTVM TEE Client Configuration\n")
	fmt.Printf("=============================\n")
	fmt.Printf("Config File: %s\n", configPath)
	fmt.Printf("\nServer Configuration:\n")
	fmt.Printf("  Address: %s\n", config.Server.Address)
	fmt.Printf("  Timeout: %d seconds\n", config.Server.Timeout)

	fmt.Printf("\nExecution Configuration:\n")
	fmt.Printf("  Version: %s\n", config.Execution.Version)
	fmt.Printf("  Request ID: %s\n", config.Execution.RequestId)
	fmt.Printf("  Function: %s\n", config.Execution.FnName)
	fmt.Printf("  Inputs: %v\n", config.Execution.Inputs)
	fmt.Printf("  Timestamp: %d (%s)\n", config.Execution.Timestamp,
		time.Unix(config.Execution.Timestamp, 0).Format(time.RFC3339))

	if config.Execution.BytecodeFile != "" {
		fmt.Printf("  Bytecode File: %s\n", config.Execution.BytecodeFile)
	} else if config.Execution.Bytecode != "" {
		fmt.Printf("  Bytecode: %s... (%d chars)\n",
			truncateString(config.Execution.Bytecode, 50), len(config.Execution.Bytecode))
	} else {
		fmt.Printf("  Bytecode: <default sample>\n")
	}

	fmt.Printf("\nRuntime Configuration:\n")
	fmt.Printf("  Mode: %s\n", config.RuntimeConfig.Mode)
	fmt.Printf("  Gas Limit: enabled=%t, limit=%d\n",
		config.RuntimeConfig.GasLimit.UseGasLimit, config.RuntimeConfig.GasLimit.Limit)
	fmt.Printf("\n")
}

// displayGRPCRequest displays the gRPC request that would be sent
func displayGRPCRequest(request *dtvm.DTVMExecutionRequest) {
	fmt.Printf("gRPC Request Structure:\n")
	fmt.Printf("======================\n")

	fmt.Printf("DTVMExecutionRequest:\n")
	fmt.Printf("  Execution:\n")
	fmt.Printf("    Version: %s\n", request.Execution.Version)
	fmt.Printf("    RequestId: %s\n", request.Execution.RequestId)
	fmt.Printf("    FnName: %s\n", request.Execution.FnName)
	fmt.Printf("    Inputs: [%d items]\n", len(request.Execution.Inputs))
	for i, input := range request.Execution.Inputs {
		decoded, _ := base64.StdEncoding.DecodeString(input)
		fmt.Printf("      [%d]: %s (base64: %s)\n", i, string(decoded), input)
	}
	fmt.Printf("    Bytecode: %s... (%d chars)\n",
		truncateString(request.Execution.Bytecode, 50), len(request.Execution.Bytecode))
	fmt.Printf("    Timestamp: %d\n", request.Execution.Timestamp)

	fmt.Printf("  RuntimeConfig:\n")
	fmt.Printf("    Mode: %s\n", request.RuntimeConfig.Mode.String())
	fmt.Printf("    GasLimit:\n")
	fmt.Printf("      UseGasLimit: %t\n", request.RuntimeConfig.GasLimit.UseGasLimit)
	fmt.Printf("      Limit: %d\n", request.RuntimeConfig.GasLimit.Limit)
}

// executeRequest executes the DTVM request based on configuration
func executeRequest(config *Config) error {
	// Create connection to gRPC server
	fmt.Printf("Connecting to server...\n")
	conn, err := grpc.Dial(config.Server.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	// Create gRPC client
	client := dtvm.NewDTVMTeeServiceClient(conn)

	// Build gRPC request from config
	request, err := config.ToGRPCRequest()
	if err != nil {
		return fmt.Errorf("failed to build request: %v", err)
	}

	// Execute request with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Server.Timeout)*time.Second)
	defer cancel()

	fmt.Printf("Executing request...\n")
	fmt.Printf("Request ID: %s\n", request.Execution.RequestId)

	startTime := time.Now()
	response, err := client.Execute(ctx, request)
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("execution failed: %v", err)
	}

	// Display results
	displayResults(response, duration)
	return nil
}

// displayResults displays the execution results
func displayResults(response *dtvm.DTVMExecutionResponse, duration time.Duration) {
	fmt.Printf("\n=== Execution Results ===\n")
	fmt.Printf("Execution Time: %v\n", duration)
	fmt.Printf("Request ID: %s\n", response.RequestId)

	if response.Result == nil {
		fmt.Printf("No result returned\n")
		return
	}

	fmt.Printf("Input Count: %d\n", len(response.Result.Inputs))
	fmt.Printf("Output Count: %d\n", len(response.Result.OutputValues))

	// Display input values
	fmt.Printf("\nInputs:\n")
	for i, input := range response.Result.Inputs {
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			fmt.Printf("  [%d]: %s (decode error: %v)\n", i, input, err)
		} else {
			fmt.Printf("  [%d]: %s\n", i, string(decoded))
		}
	}

	// Display output values
	fmt.Printf("\nOutputs:\n")
	for i, value := range response.Result.OutputValues {
		fmt.Printf("  [%d]: Type=%s, ", i, value.Type.String())
		switch value.Value.(type) {
		case *dtvm.Value_Int32Value:
			fmt.Printf("Value=%d\n", value.GetInt32Value())
		case *dtvm.Value_Int64Value:
			fmt.Printf("Value=%d\n", value.GetInt64Value())
		case *dtvm.Value_Float32Value:
			fmt.Printf("Value=%f\n", value.GetFloat32Value())
		case *dtvm.Value_Float64Value:
			fmt.Printf("Value=%f\n", value.GetFloat64Value())
		default:
			fmt.Printf("Value=unknown type\n")
		}
	}

	// Display attestation info
	fmt.Printf("\nAttestation:\n")
	fmt.Printf("  Data Length: %d bytes\n", len(response.Result.Attestation))
	if len(response.Result.Attestation) > 0 {
		fmt.Printf("  Preview: %s...\n", truncateString(response.Result.Attestation, 100))
	}

	// Display report data
	fmt.Printf("\nReport Data:\n")
	fmt.Printf("  Hash: %s\n", response.Result.ReportData)

	fmt.Printf("\n✅ Execution completed successfully!\n")
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// createSampleBytecode creates a minimal WASM module with an add function
func createSampleBytecode() string {
	// This is a minimal WASM module that exports an "add" function
	// The function takes two i32 parameters and returns their sum
	wasmBytes := []byte{
		0x00, 0x61, 0x73, 0x6d, // WASM magic number
		0x01, 0x00, 0x00, 0x00, // WASM version
		// Type section: function signature (i32, i32) -> i32
		0x01, 0x07, 0x01, 0x60, 0x02, 0x7f, 0x7f, 0x01, 0x7f,
		// Function section: one function of type 0
		0x03, 0x02, 0x01, 0x00,
		// Export section: export function 0 as "add"
		0x07, 0x07, 0x01, 0x03, 0x61, 0x64, 0x64, 0x00, 0x00,
		// Code section: function body
		0x0a, 0x09, 0x01, 0x07, 0x00, 0x20, 0x00, 0x20, 0x01, 0x6a, 0x0b,
	}
	return base64.StdEncoding.EncodeToString(wasmBytes)
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

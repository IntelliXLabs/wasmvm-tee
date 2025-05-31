package wasm

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// These are placeholder constants. Replace them with the actual values
// or enum members from the wasmedge-bindgen package if they are not globally defined.
const (
	U8 uint32 = iota // Placeholder, replace with actual value
	I8
	U16
	I16
	U32
	I32
	U64
	I64
	F32
	F64
	Bool
	Rune // Note: wasmedge-bindgen might treat rune as i32 or similar.
	String
	ByteArray
	I8Array
	U16Array
	I16Array
	U32Array
	I32Array
	U64Array
	I64Array
)

// calculateStandardHash provides a standardized way to hash protobuf messages
// External systems can use the same method for verification
func (s *Server) calculateStandardHash(messages ...proto.Message) ([32]byte, error) {
	var allData []byte

	for i, msg := range messages {
		// Add message index for ordering
		indexBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(indexBytes, uint32(i))
		allData = append(allData, indexBytes...)

		// Serialize message
		data, err := proto.Marshal(msg)
		if err != nil {
			return [32]byte{}, fmt.Errorf("failed to marshal message %d: %v", i, err)
		}

		// Add length prefix for clear separation
		lengthBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lengthBytes, uint32(len(data)))
		allData = append(allData, lengthBytes...)
		allData = append(allData, data...)
	}

	return sha256.Sum256(allData), nil
}

// calculateOutputHash wraps output values for hash calculation
func (s *Server) calculateOutputHash(outputs []*WasmValue) ([32]byte, error) {
	messages := make([]proto.Message, len(outputs))
	for i, v := range outputs {
		messages[i] = v
	}

	return s.calculateStandardHash(messages...)
}

func ConvertWasmValuesToInterface(input []*WasmValue) ([]interface{}, error) {
	if input == nil {
		return nil, fmt.Errorf("input WasmValue is nil")
	}

	results := make([]interface{}, len(input))
	for i, v := range input {
		converted, err := ConvertWasmValueToInterface(v)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input at index %d: %v", i, err)
		}
		results[i] = converted
	}

	return results, nil
}

// ConvertWasmValueToInterface converts an WasmValue protobuf message to a Go interface{}.
// This interface{} can then be used as a parameter for wasmedge-bindgen's Execute function.
func ConvertWasmValueToInterface(input *WasmValue) (interface{}, error) {
	if input == nil {
		return nil, fmt.Errorf("input WasmValue is nil")
	}

	switch v := input.Value.(type) {
	// Basic Numeric Types
	case *WasmValue_BoolValue:
		return v.BoolValue, nil
	case *WasmValue_Int8Value:
		// Protobuf stores this as int32, so we cast it back to int8.
		// It's crucial that the value was originally an int8.
		return int8(v.Int8Value), nil
	case *WasmValue_Uint8Value:
		// Protobuf stores this as uint32, cast back to uint8.
		return uint8(v.Uint8Value), nil
	case *WasmValue_Int16Value:
		// Protobuf stores this as int32, cast back to int16.
		return int16(v.Int16Value), nil
	case *WasmValue_Uint16Value:
		// Protobuf stores this as uint32, cast back to uint16.
		return uint16(v.Uint16Value), nil
	case *WasmValue_Int32Value:
		return v.Int32Value, nil
	case *WasmValue_Uint32Value:
		return v.Uint32Value, nil
	case *WasmValue_Int64Value:
		return v.Int64Value, nil
	case *WasmValue_Uint64Value:
		return v.Uint64Value, nil
	case *WasmValue_Float32Value:
		return v.Float32Value, nil
	case *WasmValue_Float64Value:
		return v.Float64Value, nil

	// String and Byte Array
	case *WasmValue_StringValue:
		return v.StringValue, nil
	case *WasmValue_BytesValue:
		return v.BytesValue, nil

	// Array Types
	case *WasmValue_Int8Array:
		if v.Int8Array == nil {
			return []int8{}, nil // Return empty slice if nil
		}
		arr := make([]int8, len(v.Int8Array.Values))
		for i, val := range v.Int8Array.Values {
			// Add range check if necessary, though protobuf validation should handle it.
			arr[i] = int8(val)
		}
		return arr, nil
	case *WasmValue_Uint16Array:
		if v.Uint16Array == nil {
			return []uint16{}, nil
		}
		arr := make([]uint16, len(v.Uint16Array.Values))
		for i, val := range v.Uint16Array.Values {
			arr[i] = uint16(val)
		}
		return arr, nil
	case *WasmValue_Int16Array:
		if v.Int16Array == nil {
			return []int16{}, nil
		}
		arr := make([]int16, len(v.Int16Array.Values))
		for i, val := range v.Int16Array.Values {
			arr[i] = int16(val)
		}
		return arr, nil
	case *WasmValue_Uint32Array:
		if v.Uint32Array == nil {
			return []uint32{}, nil
		}
		return v.Uint32Array.Values, nil // Direct use as it's already []uint32
	case *WasmValue_Int32Array:
		if v.Int32Array == nil {
			return []int32{}, nil
		}
		return v.Int32Array.Values, nil // Direct use
	case *WasmValue_Uint64Array:
		if v.Uint64Array == nil {
			return []uint64{}, nil
		}
		return v.Uint64Array.Values, nil // Direct use
	case *WasmValue_Int64Array:
		if v.Int64Array == nil {
			return []int64{}, nil
		}
		return v.Int64Array.Values, nil // Direct use
	default:
		return nil, fmt.Errorf("unsupported input value type: %T", v)
	}
}

// ConvertBindgenResultToWasmValues converts a slice of interfaces (results from wasmedge-bindgen)
// into a slice of *WasmValue protobuf messages.
// The `bindgenTypes` map is used to determine how to interpret each interface{} value
// based on the type identifier that wasmedge-bindgen internally uses.
// The `bindgenRawResults` is the []interface{} returned by bindgen's Execute or parse_result.
// The `bindgenTypeIdentifiers` would be a parallel slice or map indicating the type of each element
// in `bindgenRawResults`. The provided `parse_result` example shows `rets[i*3+1]` as this identifier.
//
// For simplicity, this example assumes `bindgenRawResults` contains elements whose types
// can be directly inferred or you have a way to know their intended bindgen type.
// A more robust solution would require the type identifier for each result element,
// similar to `rets[i*3+1]` in your `parse_result` example.
//
// This function focuses on converting the final Go typed values from bindgen
// (e.g., a Go string, a Go []int8) into the WasmValue protobuf structure.
func ConvertBindgenExecuteResultToWasmValues(bindgenResults []interface{}) ([]*WasmValue, error) {
	if bindgenResults == nil {
		return []*WasmValue{}, nil
	}

	wasmValues := make([]*WasmValue, len(bindgenResults))
	var err error

	for i, result := range bindgenResults {
		if result == nil {
			// Decide how to represent nil. Maybe a specific WasmValue type or skip.
			// For now, skipping or returning an error might be best.
			// wasmValues[i] = nil // Or some default WasmValue
			return nil, fmt.Errorf("nil result at index %d cannot be converted", i)
		}

		switch v := result.(type) {
		case bool:
			wasmValues[i] = &WasmValue{Value: &WasmValue_BoolValue{BoolValue: v}}
		case int8:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int8Value{Int8Value: int32(v)}} // Protobuf uses int32 for int8
		case uint8:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Uint8Value{Uint8Value: uint32(v)}} // Protobuf uses uint32 for uint8
		case int16:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int16Value{Int16Value: int32(v)}} // Protobuf uses int32 for int16
		case uint16:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Uint16Value{Uint16Value: uint32(v)}} // Protobuf uses uint32 for uint16
		case int32:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int32Value{Int32Value: v}}
		case uint32:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Uint32Value{Uint32Value: v}}
		case int64:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int64Value{Int64Value: v}}
		case uint64:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Uint64Value{Uint64Value: v}}
		case float32:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Float32Value{Float32Value: v}}
		case float64:
			wasmValues[i] = &WasmValue{Value: &WasmValue_Float64Value{Float64Value: v}}
		case string:
			wasmValues[i] = &WasmValue{Value: &WasmValue_StringValue{StringValue: v}}
		case []byte: // This covers ByteArray
			wasmValues[i] = &WasmValue{Value: &WasmValue_BytesValue{BytesValue: v}}
		case []int8: // This covers I8Array
			// Convert []int8 to []int32 for protobuf
			values := make([]int32, len(v))
			for j, item := range v {
				values[j] = int32(item)
			}
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int8Array{Int8Array: &Int8Array{Values: values}}}
		case []uint16: // This covers U16Array
			values := make([]uint32, len(v))
			for j, item := range v {
				values[j] = uint32(item)
			}
			wasmValues[i] = &WasmValue{Value: &WasmValue_Uint16Array{Uint16Array: &Uint16Array{Values: values}}}
		case []int16: // This covers I16Array
			values := make([]int32, len(v))
			for j, item := range v {
				values[j] = int32(item)
			}
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int16Array{Int16Array: &Int16Array{Values: values}}}
		case []uint32: // This covers U32Array
			wasmValues[i] = &WasmValue{Value: &WasmValue_Uint32Array{Uint32Array: &Uint32Array{Values: v}}}
		case []int32: // This covers I32Array
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int32Array{Int32Array: &Int32Array{Values: v}}}
		case []uint64: // This covers U64Array
			wasmValues[i] = &WasmValue{Value: &WasmValue_Uint64Array{Uint64Array: &Uint64Array{Values: v}}}
		case []int64: // This covers I64Array
			wasmValues[i] = &WasmValue{Value: &WasmValue_Int64Array{Int64Array: &Int64Array{Values: v}}}
		// Note: wasmedge-bindgen's 'Rune' type might be an alias for int32.
		// If it's a distinct type, you'd need a case for it.
		// case rune:
		// wasmValues[i] = &WasmValue{Value: &WasmValue_Int32Value{Int32Value: int32(v)}}
		default:
			return nil, fmt.Errorf("unsupported type in bindgen result at index %d: %T", i, result)
		}
	}

	return wasmValues, err
}

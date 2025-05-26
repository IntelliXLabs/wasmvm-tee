package dtvm

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"google.golang.org/protobuf/proto"
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
func (s *Server) calculateOutputHash(outputs []*Value) ([32]byte, error) {
	messages := make([]proto.Message, len(outputs))
	for i, v := range outputs {
		messages[i] = v
	}

	return s.calculateStandardHash(messages...)
}

// encodeStringsToBase64 converts string array to base64 encoded string array
// Used for encoding input parameters in the execution result
func encodeStringsToBase64(strs []string) []string {
	encoded := make([]string, len(strs))
	for i, str := range strs {
		encoded[i] = base64.StdEncoding.EncodeToString([]byte(str))
	}
	return encoded
}

package dtvm

import (
	"fmt"

	"github.com/google/go-sev-guest/client"
	"google.golang.org/protobuf/encoding/protojson"
)

func generateAttestation(reportData [64]byte) ([]byte, error) {
	// Get quote provider instead of opening device directly
	quoteProvider, err := client.GetQuoteProvider()
	if err != nil {
		return nil, fmt.Errorf("getting quote provider: %w", err)
	}

	// Get attestation report using quote provider
	attestation, err := client.GetQuoteProto(quoteProvider, reportData)
	if err != nil {
		return nil, fmt.Errorf("getting attestation report: %w", err)
	}

	// Convert attestation to JSON string
	jsonBytes, err := protojson.Marshal(attestation)
	if err != nil {
		return nil, fmt.Errorf("marshaling attestation to JSON: %w", err)
	}

	return jsonBytes, nil
}

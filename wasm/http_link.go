package wasm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/second-state/WasmEdge-go/wasmedge"
)

// HttpRequest represents a complete HTTP request structure
type HttpRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Timeout int               `json:"timeout,omitempty"` // timeout in seconds
}

// HttpResponse represents the HTTP response
type HttpResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
}

// performHttpRequest performs a complete HTTP request with full control
func performHttpRequest(requestJSON string) []byte {
	var httpReq HttpRequest
	if err := json.Unmarshal([]byte(requestJSON), &httpReq); err != nil {
		response := HttpResponse{
			StatusCode: 0,
			Error:      fmt.Sprintf("Failed to parse request JSON: %v", err),
		}
		respJSON, _ := json.Marshal(response)
		return respJSON
	}

	// Set default values
	if httpReq.Method == "" {
		httpReq.Method = "GET"
	}
	if httpReq.Timeout == 0 {
		httpReq.Timeout = 30 // default 30 seconds
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: time.Duration(httpReq.Timeout) * time.Second,
	}

	// Create request body
	var reqBody io.Reader
	if httpReq.Body != "" {
		reqBody = strings.NewReader(httpReq.Body)
	}

	// Create HTTP request
	req, err := http.NewRequest(strings.ToUpper(httpReq.Method), httpReq.URL, reqBody)
	if err != nil {
		response := HttpResponse{
			StatusCode: 0,
			Error:      fmt.Sprintf("Failed to create request: %v", err),
		}

		respJSON, _ := json.Marshal(response)
		return respJSON
	}

	// Set headers
	for key, value := range httpReq.Headers {
		req.Header.Set(key, value)
	}

	// Set default Content-Type for POST/PUT/PATCH requests with body
	if httpReq.Body != "" && req.Header.Get("Content-Type") == "" {
		if strings.ToUpper(httpReq.Method) == "POST" || strings.ToUpper(httpReq.Method) == "PUT" || strings.ToUpper(httpReq.Method) == "PATCH" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Perform the request
	resp, err := client.Do(req)
	if err != nil {
		response := HttpResponse{
			StatusCode: 0,
			Error:      fmt.Sprintf("Request failed: %v", err),
		}
		respJSON, _ := json.Marshal(response)
		return respJSON
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		response := HttpResponse{
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("Failed to read response body: %v", err),
		}
		respJSON, _ := json.Marshal(response)
		return respJSON
	}

	// Extract response headers
	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[key] = values[0] // Take the first value if multiple
		}
	}

	// Create response
	response := HttpResponse{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(body),
	}

	respJSON, _ := json.Marshal(response)
	return respJSON
}

// Host function for fetching - now supports complete HTTP requests
func (h *host) http(_ any, callframe *wasmedge.CallingFrame, params []any) ([]any, wasmedge.Result) {
	// get request JSON from memory
	pointer := params[0].(int32)
	size := params[1].(int32)
	mem := callframe.GetMemoryByIndex(0)
	data, _ := mem.GetData(uint(pointer), uint(size))
	requestData := make([]byte, size)
	copy(requestData, data)

	requestStr := string(requestData)

	// Try to parse as JSON first (new format), fallback to simple URL (legacy)
	var respBody []byte
	var httpReq HttpRequest
	if err := json.Unmarshal([]byte(requestStr), &httpReq); err == nil {
		// New format: complete HTTP request JSON
		respBody = performHttpRequest(requestStr)
	} else {
		// Legacy format: simple URL string
		legacyResp := fetch(requestStr)
		if legacyResp == nil {
			return nil, wasmedge.Result_Fail
		}
		respBody = legacyResp
	}

	if respBody == nil {
		return nil, wasmedge.Result_Fail
	}

	// store the response
	h.fetchResult = respBody

	return []any{any(len(respBody))}, wasmedge.Result_Success
}

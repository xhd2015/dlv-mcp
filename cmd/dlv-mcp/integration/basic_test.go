package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	basicServerPort = "2280" // Different port from the main test
	basicServerAddr = "http://localhost:" + basicServerPort
)

// jsonRPCRequest represents a JSON-RPC request to send to the MCP server
type basicJSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCResponse represents a JSON-RPC response from the MCP server
type basicJSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// TestSSEBasicConnectivity tests just the basic connectivity to the SSE server
// This test assumes the server is already running with:
// go run ./cmd/dlv-mcp --listen :2280
func TestSSEBasicConnectivity(t *testing.T) {
	// Connect to the SSE endpoint
	t.Logf("Connecting to SSE endpoint at %s", basicServerAddr+"/sse")
	resp, err := http.Get(basicServerAddr + "/sse")
	require.NoError(t, err, "Failed to connect to SSE endpoint")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status from SSE endpoint")
	t.Logf("Successfully connected to SSE endpoint with status %d", resp.StatusCode)

	// Read the first event to verify we get an endpoint
	scanner := bufio.NewScanner(resp.Body)
	var foundEndpoint bool
	var endpoint string

	for scanner.Scan() {
		line := scanner.Text()
		t.Logf("SSE: %s", line)

		if line == "" {
			// End of event
			break
		}

		if strings.HasPrefix(line, "event: endpoint") {
			foundEndpoint = true
			// Next line should contain the data
			if scanner.Scan() {
				data := scanner.Text()
				t.Logf("SSE: %s", data)
				if strings.HasPrefix(data, "data: ") {
					endpoint = strings.TrimPrefix(data, "data: ")
					t.Logf("Found endpoint: %s", endpoint)
				}
			}
			break
		}
	}

	require.True(t, foundEndpoint, "Should have received an 'endpoint' event")
	require.NotEmpty(t, endpoint, "Should have received a non-empty endpoint URL")

	// Now let's send a JSON-RPC request to the message endpoint
	// If the endpoint is relative, prepend the server address
	fullEndpoint := endpoint
	if strings.HasPrefix(endpoint, "/") {
		fullEndpoint = basicServerAddr + endpoint
	}
	t.Logf("Using full message endpoint: %s", fullEndpoint)

	// Create an initialize request
	initRequest := basicJSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"client": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
			"capabilities": map[string]interface{}{
				"resources": map[string]interface{}{
					"subscribe": true,
				},
			},
		},
	}

	// Send the request
	reqBody, err := json.Marshal(initRequest)
	require.NoError(t, err, "Failed to marshal request")

	httpReq, err := http.NewRequest("POST", fullEndpoint, bytes.NewReader(reqBody))
	require.NoError(t, err, "Failed to create HTTP request")
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	require.NoError(t, err, "Failed to send request")
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err, "Failed to read response body")

	t.Logf("Initialize response status: %d", httpResp.StatusCode)
	t.Logf("Initialize response body: %s", string(respBody))

	// Check response status
	require.True(t, httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusAccepted,
		"Expected status 200 OK or 202 Accepted, got %d", httpResp.StatusCode)

	// Parse the response
	var initResponse basicJSONRPCResponse
	err = json.Unmarshal(respBody, &initResponse)
	require.NoError(t, err, "Failed to unmarshal response")

	// Verify there's no error
	require.Nil(t, initResponse.Error, "Initialize request returned error: %v", initResponse.Error)

	// Verify there's a result
	require.NotNil(t, initResponse.Result, "Expected non-nil result")

	// Try to parse and pretty-print the result
	var resultMap map[string]interface{}
	err = json.Unmarshal(initResponse.Result, &resultMap)
	if err == nil {
		resultJSON, _ := json.MarshalIndent(resultMap, "", "  ")
		t.Logf("Initialize result: %s", string(resultJSON))
	}
}

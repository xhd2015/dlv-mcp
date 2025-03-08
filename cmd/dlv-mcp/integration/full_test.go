package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fullServerPort = "2279" // Port used by the main server
	fullServerAddr = "http://localhost:" + fullServerPort
)

// jsonRPCRequest represents a JSON-RPC request to send to the MCP server
type fullJSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCResponse represents a JSON-RPC response from the MCP server
type fullJSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// TestFullSSEServer is a comprehensive integration test for the SSE server
// It tests the full SSE-based communication flow with the MCP server:
//  1. Start the MCP server with SSE support
//  2. Connect to the SSE endpoint to establish a server-sent events connection
//  3. Get the message endpoint with session ID from the SSE connection
//  4. Send JSON-RPC requests to the message endpoint using the session ID
//
// Authentication works as follows:
//   - The client connects to /sse to establish an SSE connection
//   - The server responds with a "endpoint" event containing a message endpoint URL with a session ID
//   - All subsequent JSON-RPC requests must be sent to this endpoint URL with the session ID
//   - The session ID is used to authenticate the client with the server
func TestFullSSEServer(t *testing.T) {
	// Start the server
	cmd := exec.Command("go", "run", "../../cmd/dlv-mcp", "--listen", ":"+fullServerPort)
	cmd.Dir = findProjectRoot(t)

	// Set up pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "Failed to create stdout pipe")
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err, "Failed to create stderr pipe")

	// Start the server
	err = cmd.Start()
	require.NoError(t, err, "Failed to start server")

	// Clean up when done
	defer func() {
		// Try graceful shutdown first
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			// Give it a moment to clean up
			time.Sleep(100 * time.Millisecond)
			// Force kill if still running
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// Read server output in background
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
		for scanner.Scan() {
			t.Logf("Server: %s", scanner.Text())
		}
	}()

	// Wait for the server to start by polling the SSE endpoint
	serverReady := false
	startTime := time.Now()
	for time.Since(startTime) < 5*time.Second {
		resp, err := http.Get(fullServerAddr + "/sse")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				serverReady = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, serverReady, "Server failed to start within 5 seconds")

	// Connect to the SSE endpoint
	t.Logf("Connecting to SSE endpoint at %s", fullServerAddr+"/sse")
	resp, err := http.Get(fullServerAddr + "/sse")
	require.NoError(t, err, "Failed to connect to SSE endpoint")
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK status from SSE endpoint")
	t.Logf("Successfully connected to SSE endpoint with status %d", resp.StatusCode)

	// Read the first event to get the message endpoint
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

	// If the endpoint is relative, prepend the server address
	fullEndpoint := endpoint
	if strings.HasPrefix(endpoint, "/") {
		fullEndpoint = fullServerAddr + endpoint
	}
	t.Logf("Using full message endpoint: %s", fullEndpoint)

	// Run initialization sequence
	t.Run("Initialize", func(t *testing.T) {
		// Create an initialize request
		initRequest := fullJSONRPCRequest{
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
		var initResponse fullJSONRPCResponse
		err = json.Unmarshal(respBody, &initResponse)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify there's no error
		require.Nil(t, initResponse.Error, "Initialize request returned error: %v", initResponse.Error)

		// Verify there's a result
		require.NotNil(t, initResponse.Result, "Expected non-nil result")

		// Parse and verify the result
		var result map[string]interface{}
		err = json.Unmarshal(initResponse.Result, &result)
		require.NoError(t, err, "Failed to unmarshal result")

		// Log the result for debugging
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("Initialize result: %s", string(resultJSON))

		// Check for server info
		serverInfo, ok := result["serverInfo"].(map[string]interface{})
		assert.True(t, ok, "Server info missing")
		assert.NotEmpty(t, serverInfo["name"], "Server name missing")
		assert.NotEmpty(t, serverInfo["version"], "Server version missing")

		// Check for protocol version
		assert.NotEmpty(t, result["protocolVersion"], "Protocol version missing")
	})

	// Test listing tools
	t.Run("ListTools", func(t *testing.T) {
		// Create tools/list request
		toolsRequest := fullJSONRPCRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools/list",
			Params:  map[string]interface{}{},
		}

		// Send the request
		reqBody, err := json.Marshal(toolsRequest)
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

		t.Logf("ListTools response status: %d", httpResp.StatusCode)
		t.Logf("ListTools response body: %s", string(respBody))

		// Check response status
		require.True(t, httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusAccepted,
			"Expected status 200 OK or 202 Accepted, got %d", httpResp.StatusCode)

		// Parse the response
		var toolsResponse fullJSONRPCResponse
		err = json.Unmarshal(respBody, &toolsResponse)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify there's no error
		require.Nil(t, toolsResponse.Error, "tools/list request returned error: %v", toolsResponse.Error)

		// Verify there's a result
		require.NotNil(t, toolsResponse.Result, "Expected non-nil result")

		// Parse the result
		var result map[string]interface{}
		err = json.Unmarshal(toolsResponse.Result, &result)
		require.NoError(t, err, "Failed to unmarshal result")

		// Log the result for debugging
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		t.Logf("ListTools result: %s", string(resultJSON))

		// Verify that tools is a non-empty array
		tools, ok := result["tools"].([]interface{})
		assert.True(t, ok, "Tools missing in response")
		assert.NotEmpty(t, tools, "Tools array is empty")

		// If we've gotten this far, log the number of tools found
		t.Logf("Found %d tools", len(tools))

		// Verify at least some debugging tools are present
		debugTools := []string{
			"start_debug",
			"terminate_debug",
			"list_debug_sessions",
			"set_breakpoint",
			"continue",
		}

		foundTools := make(map[string]bool)
		for _, tool := range tools {
			toolMap, ok := tool.(map[string]interface{})
			if !ok {
				continue
			}
			if name, ok := toolMap["name"].(string); ok {
				foundTools[name] = true
				t.Logf("Found tool: %s", name)
			}
		}

		for _, toolName := range debugTools {
			assert.True(t, foundTools[toolName], "Expected debug tool missing: %s", toolName)
		}
	})

	// Test ping
	t.Run("Ping", func(t *testing.T) {
		// Create ping request
		pingRequest := fullJSONRPCRequest{
			JSONRPC: "2.0",
			ID:      3,
			Method:  "ping",
			Params:  map[string]interface{}{},
		}

		// Send the request
		reqBody, err := json.Marshal(pingRequest)
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

		t.Logf("Ping response status: %d", httpResp.StatusCode)
		t.Logf("Ping response body: %s", string(respBody))

		// Check response status
		require.True(t, httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusAccepted,
			"Expected status 200 OK or 202 Accepted, got %d", httpResp.StatusCode)

		// Parse the response
		var pingResponse fullJSONRPCResponse
		err = json.Unmarshal(respBody, &pingResponse)
		require.NoError(t, err, "Failed to unmarshal response")

		// Verify there's no error
		require.Nil(t, pingResponse.Error, "ping request returned error: %v", pingResponse.Error)

		// Verify there's a result (might be empty object)
		require.NotNil(t, pingResponse.Result, "Expected non-nil result from ping")
	})
}

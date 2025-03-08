package integration

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	remoteServerPort = "2281" // Different port from other tests
	remoteServerAddr = "http://localhost:" + remoteServerPort
	dlvPort          = "2345" // Port for headless dlv
)

// TestRemoteDebugger tests the remote debugging functionality
func TestRemoteDebugger(t *testing.T) {
	// Find project root
	projectRoot := findProjectRoot(t)

	// Create a simple Go program to debug
	debugProgramDir := filepath.Join(projectRoot, "tmp", "remote_debug_test")
	err := os.MkdirAll(debugProgramDir, 0755)
	require.NoError(t, err, "Failed to create test directory")
	defer os.RemoveAll(debugProgramDir)

	debugProgram := filepath.Join(debugProgramDir, "main.go")
	err = os.WriteFile(debugProgram, []byte(`package main

import "fmt"

func main() {
	x := 42
	fmt.Println("Hello from debuggee")
	fmt.Printf("x = %d\n", x)
}
`), 0644)
	require.NoError(t, err, "Failed to write test program")

	// Start headless dlv
	dlvCmd := exec.Command("dlv", "debug", "--headless", "--listen=:"+dlvPort, "--api-version=2", "--accept-multiclient")
	dlvCmd.Dir = debugProgramDir
	dlvStdout, err := dlvCmd.StdoutPipe()
	require.NoError(t, err, "Failed to create dlv stdout pipe")
	dlvStderr, err := dlvCmd.StderrPipe()
	require.NoError(t, err, "Failed to create dlv stderr pipe")

	err = dlvCmd.Start()
	require.NoError(t, err, "Failed to start dlv")

	// Clean up dlv when done
	defer func() {
		if dlvCmd.Process != nil {
			dlvCmd.Process.Signal(os.Interrupt)
			time.Sleep(100 * time.Millisecond)
			dlvCmd.Process.Kill()
			dlvCmd.Wait()
		}
	}()

	// Read dlv output in background
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(dlvStdout, dlvStderr))
		for scanner.Scan() {
			t.Logf("DLV: %s", scanner.Text())
		}
	}()

	// Start the MCP server
	mcpCmd := exec.Command("go", "run", "../../cmd/dlv-mcp", "--listen", ":"+remoteServerPort)
	mcpCmd.Dir = projectRoot

	mcpStdout, err := mcpCmd.StdoutPipe()
	require.NoError(t, err, "Failed to create mcp stdout pipe")
	mcpStderr, err := mcpCmd.StderrPipe()
	require.NoError(t, err, "Failed to create mcp stderr pipe")

	err = mcpCmd.Start()
	require.NoError(t, err, "Failed to start mcp server")

	// Clean up MCP when done
	defer func() {
		if mcpCmd.Process != nil {
			mcpCmd.Process.Signal(os.Interrupt)
			time.Sleep(100 * time.Millisecond)
			mcpCmd.Process.Kill()
			mcpCmd.Wait()
		}
	}()

	// Read MCP output in background
	go func() {
		scanner := bufio.NewScanner(io.MultiReader(mcpStdout, mcpStderr))
		for scanner.Scan() {
			t.Logf("MCP: %s", scanner.Text())
		}
	}()

	// Wait for the server to start
	serverReady := false
	startTime := time.Now()
	for time.Since(startTime) < 5*time.Second {
		resp, err := http.Get(remoteServerAddr + "/sse")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				serverReady = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.True(t, serverReady, "MCP server failed to start within 5 seconds")

	// Connect to SSE endpoint
	resp, err := http.Get(remoteServerAddr + "/sse")
	require.NoError(t, err, "Failed to connect to SSE endpoint")
	defer resp.Body.Close()

	// Get message endpoint from SSE
	scanner := bufio.NewScanner(resp.Body)
	var endpoint string
	for scanner.Scan() {
		line := scanner.Text()
		t.Logf("SSE: %s", line)

		if strings.HasPrefix(line, "event: endpoint") {
			if scanner.Scan() {
				data := scanner.Text()
				if strings.HasPrefix(data, "data: ") {
					endpoint = strings.TrimPrefix(data, "data: ")
					break
				}
			}
		}
	}
	require.NotEmpty(t, endpoint, "Failed to get message endpoint")

	// Initialize the client
	fullEndpoint := remoteServerAddr + endpoint
	initResp := sendJSONRPC(t, fullEndpoint, "initialize", map[string]interface{}{
		"client": map[string]interface{}{
			"name":    "test-client",
			"version": "1.0.0",
		},
		"capabilities": map[string]interface{}{
			"resources": map[string]interface{}{
				"subscribe": true,
			},
		},
	})
	require.NotNil(t, initResp.Result, "Initialize failed")

	// Start remote debug session
	startResp := sendJSONRPC(t, fullEndpoint, "start_debug_remote", map[string]interface{}{
		"cwd":     debugProgramDir,
		"address": fmt.Sprintf("localhost:%s", dlvPort),
	})
	require.NotNil(t, startResp.Result, "Start debug remote failed")

	// Extract session ID from response
	var sessionID string
	{
		var result map[string]interface{}
		err := json.Unmarshal(startResp.Result, &result)
		require.NoError(t, err, "Failed to parse start_debug_remote response")
		text, ok := result["text"].(string)
		require.True(t, ok, "Response missing text field")
		for _, line := range strings.Split(text, "\n") {
			if strings.HasPrefix(line, "Remote debug session started with ID: ") {
				sessionID = strings.TrimPrefix(line, "Remote debug session started with ID: ")
				break
			}
		}
	}
	require.NotEmpty(t, sessionID, "Failed to get session ID")

	// Set a breakpoint
	bpResp := sendJSONRPC(t, fullEndpoint, "set_breakpoint", map[string]interface{}{
		"session_id": sessionID,
		"file":       debugProgram,
		"line":       6, // Line with fmt.Println
	})
	require.NotNil(t, bpResp.Result, "Set breakpoint failed")

	// Continue execution
	contResp := sendJSONRPC(t, fullEndpoint, "continue", map[string]interface{}{
		"session_id": sessionID,
	})
	require.NotNil(t, contResp.Result, "Continue failed")

	// Evaluate x variable
	evalResp := sendJSONRPC(t, fullEndpoint, "evaluate", map[string]interface{}{
		"session_id": sessionID,
		"expression": "x",
	})
	require.NotNil(t, evalResp.Result, "Evaluate failed")

	// Verify x = 42
	var evalResult map[string]interface{}
	err = json.Unmarshal(evalResp.Result, &evalResult)
	require.NoError(t, err, "Failed to parse evaluate response")
	text, ok := evalResult["text"].(string)
	require.True(t, ok, "Evaluate response missing text field")
	assert.Contains(t, text, "42", "Expected x to be 42")

	// Terminate session
	termResp := sendJSONRPC(t, fullEndpoint, "terminate_debug", map[string]interface{}{
		"session_id": sessionID,
	})
	require.NotNil(t, termResp.Result, "Terminate failed")
}

// Helper function to send JSON-RPC requests
func sendJSONRPC(t *testing.T, endpoint string, method string, params interface{}) *fullJSONRPCResponse {
	t.Helper()

	reqBody, err := json.Marshal(fullJSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	require.NoError(t, err, "Failed to marshal request")

	httpReq, err := http.NewRequest("POST", endpoint, bytes.NewReader(reqBody))
	require.NoError(t, err, "Failed to create HTTP request")
	httpReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	httpResp, err := httpClient.Do(httpReq)
	require.NoError(t, err, "Failed to send request")
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	require.NoError(t, err, "Failed to read response body")

	t.Logf("%s response status: %d", method, httpResp.StatusCode)
	t.Logf("%s response body: %s", method, string(respBody))

	require.True(t, httpResp.StatusCode == http.StatusOK || httpResp.StatusCode == http.StatusAccepted,
		"Expected status 200 OK or 202 Accepted, got %d", httpResp.StatusCode)

	var response fullJSONRPCResponse
	err = json.Unmarshal(respBody, &response)
	require.NoError(t, err, "Failed to unmarshal response")
	require.Nil(t, response.Error, "%s request returned error: %v", method, response.Error)

	return &response
}

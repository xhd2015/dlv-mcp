package debug

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xhd2015/dlv-mcp/vendir/third-party/github.com/mark3labs/mcp-go/mcp"
	"github.com/xhd2015/dlv-mcp/vendir/third-party/github.com/mark3labs/mcp-go/server"
)

// TestDebugToolsRegistration tests that debug tools are registered with the server
func TestDebugToolsRegistration(t *testing.T) {

	defer func() {
		if r := recover(); r != nil {
			debug.PrintStack()
			t.Fatalf("RegisterTools function panicked: %v", r)
		}
	}()

	// Create a real MCP server
	s := server.NewMCPServer(
		"Test Server",
		"1.0.0",
	)

	// Register debug tools with the real server
	RegisterTools(s)

	// Success if we get here
}

// TestModeEnumDefinition verifies that the mode parameter has enum values defined
// and that the description includes information about the enum values
func TestModeEnumDefinition(t *testing.T) {
	// Create a real MCP server
	s := server.NewMCPServer(
		"Test Server",
		"1.0.0",
	)

	// Register debug tools with the real server
	RegisterTools(s)

	// Create a tool request for listing tools
	toolsReq := struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
	}{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      1,
		Method:  "tools/list",
	}

	// Marshal to JSON
	reqJSON, err := json.Marshal(toolsReq)
	require.NoError(t, err, "Failed to marshal request")

	// Send request to server
	resp := s.HandleMessage(context.Background(), reqJSON)

	// Check for success response
	jsonResp, ok := resp.(mcp.JSONRPCResponse)
	require.True(t, ok, "Unexpected response type: %T", resp)

	// Extract tools list
	toolsResult, err := json.Marshal(jsonResp.Result)
	require.NoError(t, err, "Failed to marshal result")

	// Parse tools list
	var result struct {
		Tools []struct {
			Name        string                 `json:"name"`
			InputSchema map[string]interface{} `json:"inputSchema"`
		} `json:"tools"`
	}

	err = json.Unmarshal(toolsResult, &result)
	require.NoError(t, err, "Failed to unmarshal tools")

	// Find start_debug tool
	var startDebugTool struct {
		Name        string                 `json:"name"`
		InputSchema map[string]interface{} `json:"inputSchema"`
	}

	found := false
	for _, tool := range result.Tools {
		if tool.Name == "start_debug" {
			startDebugTool = tool
			found = true
			break
		}
	}

	// Verify tool was found
	require.True(t, found, "start_debug tool not found")

	// Get properties from schema
	properties, ok := startDebugTool.InputSchema["properties"].(map[string]interface{})
	require.True(t, ok, "Failed to get properties from schema")

	// Get mode property
	mode, ok := properties["mode"].(map[string]interface{})
	require.True(t, ok, "Mode property not found in schema")

	// 1. Check for enum definition
	enum, ok := mode["enum"].([]interface{})
	require.True(t, ok, "Enum not found for mode property")

	// Verify specific enum values
	expectedEnums := []string{"debug", "test", "exec"}
	for _, expected := range expectedEnums {
		found := false
		for _, actual := range enum {
			if actualStr, ok := actual.(string); ok && actualStr == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected enum value '%s' not found", expected)
	}

	// Verify no extra values
	assert.Equal(t, len(expectedEnums), len(enum), "Unexpected number of enum values")

	// 2. Check the description
	description, ok := mode["description"].(string)
	require.True(t, ok, "Description not found for mode parameter")

	// The description should mention each mode
	for _, expectedMode := range expectedEnums {
		assert.True(t, strings.Contains(strings.ToLower(description), expectedMode),
			"Description should mention mode '%s', got: %s", expectedMode, description)
	}

	// Print result for debugging
	t.Logf("Mode parameter has correct enum values: %v", enum)
	t.Logf("Mode parameter description: %s", description)
}

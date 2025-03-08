package debug_ext

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless/headless_ext"
)

// registerSourceTools registers tools for source code inspection
func registerSourceTools(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	registerListSourcesTool(s, sessionManager, opts)
}

// registerListSourcesTool registers the list_sources tool
func registerListSourcesTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("list_sources",
		mcp.WithDescription("List source files in the program"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithString("filter",
			mcp.Description("Filter to apply to source files (e.g., '*.go' or a package path)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("list_sources: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		// Extract filter (optional)
		var filter string
		if filterVal, ok := request.Params.Arguments["filter"]; ok && filterVal != nil {
			filter, _ = filterVal.(string)
		}

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to list source files
		result, err := headless_ext.ListSources(session, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to list source files: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

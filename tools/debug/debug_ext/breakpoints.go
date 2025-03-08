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

// registerBreakpointTools registers tools for breakpoint management
func registerBreakpointTools(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	registerListBreakpointsTool(s, sessionManager, opts)
	registerToggleBreakpointTool(s, sessionManager, opts)
	registerClearBreakpointTool(s, sessionManager, opts)
	registerCreateWatchpointTool(s, sessionManager, opts)
}

// registerListBreakpointsTool registers the list_breakpoints tool
func registerListBreakpointsTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("list_breakpoints",
		mcp.WithDescription("List all breakpoints in the current debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("list_breakpoints: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to list breakpoints
		result, err := headless_ext.ListBreakpoints(session)
		if err != nil {
			return nil, fmt.Errorf("failed to list breakpoints: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerToggleBreakpointTool registers the toggle_breakpoint tool
func registerToggleBreakpointTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("toggle_breakpoint",
		mcp.WithDescription("Enable or disable a breakpoint"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithNumber("breakpoint_id",
			mcp.Required(),
			mcp.Description("ID of the breakpoint to toggle"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("toggle_breakpoint: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		breakpointIDFloat, _ := request.Params.Arguments["breakpoint_id"].(float64)
		if breakpointIDFloat <= 0 {
			return nil, fmt.Errorf("invalid breakpoint_id parameter")
		}
		breakpointID := int(breakpointIDFloat)

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to toggle the breakpoint
		result, err := headless_ext.ToggleBreakpoint(session, breakpointID)
		if err != nil {
			return nil, fmt.Errorf("failed to toggle breakpoint: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerClearBreakpointTool registers the clear_breakpoint tool
func registerClearBreakpointTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("clear_breakpoint",
		mcp.WithDescription("Remove a breakpoint"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithNumber("breakpoint_id",
			mcp.Required(),
			mcp.Description("ID of the breakpoint to remove"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("clear_breakpoint: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		breakpointIDFloat, _ := request.Params.Arguments["breakpoint_id"].(float64)
		if breakpointIDFloat <= 0 {
			return nil, fmt.Errorf("invalid breakpoint_id parameter")
		}
		breakpointID := int(breakpointIDFloat)

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to clear the breakpoint
		result, err := headless_ext.ClearBreakpoint(session, breakpointID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear breakpoint: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerCreateWatchpointTool registers the create_watchpoint tool
func registerCreateWatchpointTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("create_watchpoint",
		mcp.WithDescription("Create a watchpoint on a variable"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithString("variable",
			mcp.Required(),
			mcp.Description("Name of the variable to watch"),
		),
		mcp.WithString("scope",
			mcp.Description("Scope of the variable (default: global)"),
		),
		mcp.WithBoolean("write",
			mcp.Description("Watch for write access (default: true)"),
		),
		mcp.WithBoolean("read",
			mcp.Description("Watch for read access (default: false)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("create_watchpoint: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		variable, _ := request.Params.Arguments["variable"].(string)
		if variable == "" {
			return nil, fmt.Errorf("invalid variable parameter")
		}

		scope, _ := request.Params.Arguments["scope"].(string)
		if scope == "" {
			scope = "global" // Default scope
		}

		write := true
		if writeParam, ok := request.Params.Arguments["write"].(bool); ok {
			write = writeParam
		}

		read := false
		if readParam, ok := request.Params.Arguments["read"].(bool); ok {
			read = readParam
		}

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to create the watchpoint
		result, err := headless_ext.CreateWatchpoint(session, variable, scope, write, read)
		if err != nil {
			return nil, fmt.Errorf("failed to create watchpoint: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

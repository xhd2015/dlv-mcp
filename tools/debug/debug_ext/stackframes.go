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

// registerStackframeTools registers tools for stackframe management
func registerStackframeTools(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	registerStacktraceTool(s, sessionManager, opts)
	registerSwitchGoroutineTool(s, sessionManager, opts)
	registerSwitchThreadTool(s, sessionManager, opts)
}

// registerStacktraceTool registers the stacktrace tool
func registerStacktraceTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("stacktrace",
		mcp.WithDescription("Get the current goroutine's stack trace"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("stacktrace: %s", string(requestJson))

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

		// Use headless_ext to get stacktrace
		result, err := headless_ext.Stacktrace(session)
		if err != nil {
			return nil, fmt.Errorf("failed to get stacktrace: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerSwitchGoroutineTool registers the switch_goroutine tool
func registerSwitchGoroutineTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("switch_goroutine",
		mcp.WithDescription("Switch to a different goroutine"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithNumber("goroutine_id",
			mcp.Required(),
			mcp.Description("ID of the goroutine to switch to"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("switch_goroutine: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		goroutineIDFloat, _ := request.Params.Arguments["goroutine_id"].(float64)
		if goroutineIDFloat <= 0 {
			return nil, fmt.Errorf("invalid goroutine_id parameter")
		}
		goroutineID := int(goroutineIDFloat)

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to switch goroutine
		result, err := headless_ext.SwitchGoroutine(session, goroutineID)
		if err != nil {
			return nil, fmt.Errorf("failed to switch goroutine: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerSwitchThreadTool registers the switch_thread tool
func registerSwitchThreadTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("switch_thread",
		mcp.WithDescription("Switch to a different thread"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithNumber("thread_id",
			mcp.Required(),
			mcp.Description("ID of the thread to switch to"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("switch_thread: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		threadIDFloat, _ := request.Params.Arguments["thread_id"].(float64)
		if threadIDFloat <= 0 {
			return nil, fmt.Errorf("invalid thread_id parameter")
		}
		threadID := int(threadIDFloat)

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to switch thread
		result, err := headless_ext.SwitchThread(session, threadID)
		if err != nil {
			return nil, fmt.Errorf("failed to switch thread: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

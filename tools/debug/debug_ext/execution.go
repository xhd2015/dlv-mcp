package debug_ext

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless/headless_ext"
	"github.com/xhd2015/dlv-mcp/vendir/third-party/github.com/mark3labs/mcp-go/mcp"
	"github.com/xhd2015/dlv-mcp/vendir/third-party/github.com/mark3labs/mcp-go/server"
)

// registerExecutionTools registers tools for execution control
func registerExecutionTools(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	registerRestartTool(s, sessionManager, opts)
	registerDetachTool(s, sessionManager, opts)
	registerDisassembleTool(s, sessionManager, opts)
}

// registerRestartTool registers the restart tool
func registerRestartTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("restart",
		mcp.WithDescription("Restart the debugged process"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("restart: %s", string(requestJson))

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

		// Use headless_ext to restart process
		result, err := headless_ext.Restart(session)
		if err != nil {
			return nil, fmt.Errorf("failed to restart process: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerDetachTool registers the detach tool
func registerDetachTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("detach",
		mcp.WithDescription("Detach from the debugged process"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithBoolean("kill",
			mcp.Description("Kill the process when detaching (default: false)"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("detach: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		kill := false
		if killParam, ok := request.Params.Arguments["kill"].(bool); ok {
			kill = killParam
		}

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to detach
		result, err := headless_ext.Detach(session, kill)
		if err != nil {
			return nil, fmt.Errorf("failed to detach: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerDisassembleTool registers the disassemble tool
func registerDisassembleTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("disassemble",
		mcp.WithDescription("Disassemble the program at the current location"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithNumber("start_pc",
			mcp.Required(),
			mcp.Description("Start program counter address"),
		),
		mcp.WithNumber("end_pc",
			mcp.Required(),
			mcp.Description("End program counter address"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("disassemble: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		startPCFloat, _ := request.Params.Arguments["start_pc"].(float64)
		if startPCFloat < 0 {
			return nil, fmt.Errorf("invalid start_pc parameter")
		}
		startPC := uint64(startPCFloat)

		endPCFloat, _ := request.Params.Arguments["end_pc"].(float64)
		if endPCFloat <= startPCFloat {
			return nil, fmt.Errorf("invalid end_pc parameter")
		}
		endPC := uint64(endPCFloat)

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to disassemble
		result, err := headless_ext.Disassemble(session, startPC, endPC)
		if err != nil {
			return nil, fmt.Errorf("failed to disassemble: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

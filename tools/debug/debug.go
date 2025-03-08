package debug

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/xhd2015/debugger-mcp/debug"
	"github.com/xhd2015/debugger-mcp/debug/common"
)

var (
	sessionManager common.SessionManager
	once           sync.Once
)

type ToolOptions struct {
	DebuggerType string
}

// RegisterTools registers the debug tools with the MCP server
func RegisterTools(s *server.MCPServer, opts ToolOptions) error {
	sessionManager, createErr := debug.NewSessionManager(opts.DebuggerType)
	if createErr != nil {
		return fmt.Errorf("failed to create session manager: %v", createErr)
	}

	// Register tools
	registerStartDebugTool(s, sessionManager, opts)
	registerTerminateDebugTool(s, sessionManager)
	registerListSessionsTool(s, sessionManager)
	registerSetBreakpointTool(s, sessionManager)
	registerContinueTool(s, sessionManager)
	registerNextTool(s, sessionManager)
	registerStepInTool(s, sessionManager)
	registerStepOutTool(s, sessionManager)
	registerEvaluateTool(s, sessionManager)

	return nil
}

func getDefaultMode(program string) (string, error) {
	// if is dir, then debug
	state, err := os.Stat(program)
	if err != nil {
		return "", err
	}
	if state.IsDir() {
		return "debug", nil
	}
	if strings.HasSuffix(program, ".go") {
		if strings.HasSuffix(program, "_test.go") {
			return "test", nil
		}
		return "debug", nil
	}
	return "exec", nil
}

// registerStartDebugTool registers the start debug tool
func registerStartDebugTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("start_debug",
		mcp.WithDescription("Start a debug session for a Go program"),
		mcp.WithString("program",
			mcp.Required(),
			mcp.Description("Path to Go program to debug (absolute or relative)"),
		),
		mcp.WithArray("args",
			mcp.Description("Command line arguments for the program"),
			mcp.Items(map[string]interface{}{"type": "string"}),
		),
		mcp.WithString("mode",
			mcp.Description("Debug mode: 'debug' for normal debugging, 'test' for debugging tests, 'exec' for executing a binary"),
			mcp.Enum("debug", "test", "exec"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		program, _ := request.Params.Arguments["program"].(string)
		mode, _ := request.Params.Arguments["mode"].(string)
		if mode == "" {
			var err error
			mode, err = getDefaultMode(program)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get default mode: %v", err)), nil
			}
		}

		// Handle args parameter
		var args []string
		if argsRaw, ok := request.Params.Arguments["args"]; ok {
			if argsArray, ok := argsRaw.([]interface{}); ok {
				for _, arg := range argsArray {
					if argStr, ok := arg.(string); ok {
						args = append(args, argStr)
					}
				}
			}
		}

		// Convert relative path to absolute
		if !filepath.IsAbs(program) {
			absPath, err := filepath.Abs(program)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get absolute path: %v", err)), nil
			}
			program = absPath
		}

		// Start debug session
		session, err := sessionManager.CreateSession(ctx, program, args, mode)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start debug session: %v", err)), nil

		}

		// Return session information
		return mcp.NewToolResultText(fmt.Sprintf("Debug session started with ID: %s\nProgram: %s\nMode: %s",
			session.ID, session.ProgramPath, mode)), nil
	})
}

// registerTerminateDebugTool registers the terminate debug tool
func registerTerminateDebugTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("terminate_debug",
		mcp.WithDescription("Terminate a debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session to terminate"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)

		// Terminate debug session
		if err := sessionManager.TerminateSession(sessionID); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to terminate debug session: %v", err)), nil
		}

		// Return success
		return mcp.NewToolResultText(fmt.Sprintf("Debug session %s terminated", sessionID)), nil
	})
}

// registerListSessionsTool registers the list sessions tool
func registerListSessionsTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("list_debug_sessions",
		mcp.WithDescription("List active debug sessions"),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get sessions
		sessions := sessionManager.ListSessions()

		if len(sessions) == 0 {
			return mcp.NewToolResultText("No active debug sessions"), nil
		}

		// Format result
		result := "Active debug sessions:\n\n"
		for _, session := range sessions {
			result += fmt.Sprintf("ID: %s\nProgram: %s\nState: %s\n\n",
				session.ID, session.ProgramPath, session.State)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerSetBreakpointTool registers the set breakpoint tool
func registerSetBreakpointTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("set_breakpoint",
		mcp.WithDescription("Set a breakpoint in a debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithString("file",
			mcp.Required(),
			mcp.Description("Source file to set breakpoint in (absolute path)"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Line number to set breakpoint at"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		file, _ := request.Params.Arguments["file"].(string)
		lineFloat, _ := request.Params.Arguments["line"].(float64)
		line := int(lineFloat)

		// Get session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get debug session: %v", err)), nil
		}

		// Set breakpoint
		id, err := session.SetBreakpoint(file, line)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to set breakpoint: %v", err)), nil
		}

		// Return success
		return mcp.NewToolResultText(fmt.Sprintf("Breakpoint set at %s:%d (ID: %d)", file, line, id)), nil
	})
}

// registerContinueTool registers the continue tool
func registerContinueTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("continue",
		mcp.WithDescription("Continue execution in a debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)

		// Get session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get debug session: %v", err)), nil
		}

		// Continue execution
		if err := session.Continue(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to continue execution: %v", err)), nil
		}

		// Return success
		return mcp.NewToolResultText("Execution continued"), nil
	})
}

// registerNextTool registers the next tool
func registerNextTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("next",
		mcp.WithDescription("Step over current line in a debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)

		// Get session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get debug session: %v", err)), nil
		}

		// Step over
		if err := session.Next(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to step over: %v", err)), nil
		}

		// Return success
		return mcp.NewToolResultText("Stepped over current line"), nil
	})
}

// registerStepInTool registers the step in tool
func registerStepInTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("step_in",
		mcp.WithDescription("Step into function in a debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)

		// Get session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get debug session: %v", err)), nil
		}

		// Step in
		if err := session.StepIn(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to step in: %v", err)), nil
		}

		// Return success
		return mcp.NewToolResultText("Stepped into function"), nil
	})
}

// registerStepOutTool registers the step out tool
func registerStepOutTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("step_out",
		mcp.WithDescription("Step out of function in a debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)

		// Get session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get debug session: %v", err)), nil
		}

		// Step out
		if err := session.StepOut(); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to step out: %v", err)), nil
		}

		// Return success
		return mcp.NewToolResultText("Stepped out of function"), nil
	})
}

// registerEvaluateTool registers the evaluate tool
func registerEvaluateTool(s *server.MCPServer, sessionManager common.SessionManager) {
	tool := mcp.NewTool("evaluate",
		mcp.WithDescription("Evaluate an expression in a debug session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithString("expression",
			mcp.Required(),
			mcp.Description("Expression to evaluate"),
		),
		mcp.WithNumber("frame_id",
			mcp.Description("Stack frame ID"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		expression, _ := request.Params.Arguments["expression"].(string)
		// frameID is no longer used in our simplified interface
		// frameIDFloat, _ := request.Params.Arguments["frame_id"].(float64)
		// frameID := int(frameIDFloat)

		// Get session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get debug session: %v", err)), nil
		}

		// Evaluate expression
		result, err := session.Evaluate(expression)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to evaluate expression: %v", err)), nil
		}

		// Return result
		return mcp.NewToolResultText(fmt.Sprintf("Expression result: %s", result)), nil
	})
}

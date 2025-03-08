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

// registerVariableTools registers tools for variable inspection and manipulation
func registerVariableTools(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	registerListLocalVarsTool(s, sessionManager, opts)
	registerListFunctionArgsTool(s, sessionManager, opts)
	registerSetVariableTool(s, sessionManager, opts)
	registerExamineMemoryTool(s, sessionManager, opts)
}

// registerListLocalVarsTool registers the list_local_vars tool
func registerListLocalVarsTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("list_local_vars",
		mcp.WithDescription("List local variables in the current scope"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("list_local_vars: %s", string(requestJson))

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

		// Use headless_ext to list local variables
		result, err := headless_ext.ListLocalVars(session)
		if err != nil {
			return nil, fmt.Errorf("failed to list local variables: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerListFunctionArgsTool registers the list_function_args tool
func registerListFunctionArgsTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("list_function_args",
		mcp.WithDescription("List function arguments in the current scope"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("list_function_args: %s", string(requestJson))

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

		// Use headless_ext to list function arguments
		result, err := headless_ext.ListFunctionArgs(session)
		if err != nil {
			return nil, fmt.Errorf("failed to list function arguments: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerSetVariableTool registers the set_variable tool
func registerSetVariableTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("set_variable",
		mcp.WithDescription("Set the value of a variable"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the variable to set"),
		),
		mcp.WithString("value",
			mcp.Required(),
			mcp.Description("New value for the variable"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("set_variable: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		name, _ := request.Params.Arguments["name"].(string)
		if name == "" {
			return nil, fmt.Errorf("invalid name parameter")
		}

		value, _ := request.Params.Arguments["value"].(string)
		if value == "" {
			return nil, fmt.Errorf("invalid value parameter")
		}

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to set variable
		result, err := headless_ext.SetVariable(session, name, value)
		if err != nil {
			return nil, fmt.Errorf("failed to set variable: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerExamineMemoryTool registers the examine_memory tool
func registerExamineMemoryTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("examine_memory",
		mcp.WithDescription("Examine memory at a given address"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithString("address",
			mcp.Required(),
			mcp.Description("Memory address to examine"),
		),
		mcp.WithNumber("length",
			mcp.Required(),
			mcp.Description("Number of bytes to examine"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("examine_memory: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		address, _ := request.Params.Arguments["address"].(string)
		if address == "" {
			return nil, fmt.Errorf("invalid address parameter")
		}

		lengthFloat, _ := request.Params.Arguments["length"].(float64)
		if lengthFloat <= 0 {
			return nil, fmt.Errorf("invalid length parameter")
		}
		length := int(lengthFloat)

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to examine memory
		result, err := headless_ext.ExamineMemory(session, address, length)
		if err != nil {
			return nil, fmt.Errorf("failed to examine memory: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

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

// registerCheckpointTools registers tools for checkpoint management
func registerCheckpointTools(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	registerCreateCheckpointTool(s, sessionManager, opts)
	registerListCheckpointsTool(s, sessionManager, opts)
	registerClearCheckpointTool(s, sessionManager, opts)
}

// registerCreateCheckpointTool registers the create_checkpoint tool
func registerCreateCheckpointTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("create_checkpoint",
		mcp.WithDescription("Create a checkpoint at the current program state"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("create_checkpoint: %s", string(requestJson))

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

		// Use headless_ext to create checkpoint
		result, err := headless_ext.CreateCheckpoint(session)
		if err != nil {
			return nil, fmt.Errorf("failed to create checkpoint: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerListCheckpointsTool registers the list_checkpoints tool
func registerListCheckpointsTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("list_checkpoints",
		mcp.WithDescription("List all checkpoints"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("list_checkpoints: %s", string(requestJson))

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

		// Use headless_ext to list checkpoints
		result, err := headless_ext.ListCheckpoints(session)
		if err != nil {
			return nil, fmt.Errorf("failed to list checkpoints: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

// registerClearCheckpointTool registers the clear_checkpoint tool
func registerClearCheckpointTool(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) {
	tool := mcp.NewTool("clear_checkpoint",
		mcp.WithDescription("Remove a checkpoint"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("ID of the debug session"),
		),
		mcp.WithNumber("checkpoint_id",
			mcp.Required(),
			mcp.Description("ID of the checkpoint to remove"),
		),
	)

	s.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		requestJson, _ := json.Marshal(request)
		opts.Logger.Infof("clear_checkpoint: %s", string(requestJson))

		// Extract parameters
		sessionID, _ := request.Params.Arguments["session_id"].(string)
		if sessionID == "" {
			return nil, fmt.Errorf("invalid session_id parameter")
		}

		checkpointIDFloat, _ := request.Params.Arguments["checkpoint_id"].(float64)
		if checkpointIDFloat <= 0 {
			return nil, fmt.Errorf("invalid checkpoint_id parameter")
		}
		checkpointID := int(checkpointIDFloat)

		// Get the debug session
		session, err := sessionManager.GetSession(sessionID)
		if err != nil {
			return nil, fmt.Errorf("debug session not found: %s", sessionID)
		}

		// Use headless_ext to clear checkpoint
		result, err := headless_ext.ClearCheckpoint(session, checkpointID)
		if err != nil {
			return nil, fmt.Errorf("failed to clear checkpoint: %w", err)
		}

		return mcp.NewToolResultText(result), nil
	})
}

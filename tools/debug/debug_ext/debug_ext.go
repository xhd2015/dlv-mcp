package debug_ext

import (
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/log"
	"github.com/xhd2015/dlv-mcp/vendir/third-party/github.com/mark3labs/mcp-go/server"
)

// ToolOptions contains configuration options for the debug tools
type ToolOptions struct {
	Logger log.Logger
}

// RegisterExtendedTools registers additional debug tools with the MCP server
func RegisterExtendedTools(s *server.MCPServer, sessionManager common.SessionManager, opts ToolOptions) error {
	// Register stack frame tools
	registerStackframeTools(s, sessionManager, opts)

	// Register variable inspection tools
	registerVariableTools(s, sessionManager, opts)

	// Register breakpoint management tools
	registerBreakpointTools(s, sessionManager, opts)

	// Register execution control tools
	registerExecutionTools(s, sessionManager, opts)

	// Register checkpoint tools
	registerCheckpointTools(s, sessionManager, opts)

	// Register source code tools
	registerSourceTools(s, sessionManager, opts)

	return nil
}

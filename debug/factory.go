package debug

import (
	"fmt"

	"github.com/xhd2015/debugger-mcp/debug/common"
	"github.com/xhd2015/debugger-mcp/debug/dap"
	"github.com/xhd2015/debugger-mcp/debug/headless"
)

// NewSessionManager creates a new session manager based on the debugger type
func NewSessionManager(debuggerType string) (common.SessionManager, error) {
	switch debuggerType {
	case "dap":
		return dap.NewSessionManager(), nil
	case "headless":
		return headless.NewSessionManager(), nil
	default:
		return nil, fmt.Errorf("unsupported debugger type: %s", debuggerType)
	}
}

// NewClient creates a new debugger client based on the debugger type
func NewClient(debuggerType string) (common.DebuggerClient, error) {
	switch debuggerType {
	case "dap":
		return dap.NewClient(), nil
	case "headless":
		return headless.NewClient(), nil
	default:
		return nil, fmt.Errorf("unsupported debugger type: %s", debuggerType)
	}
}

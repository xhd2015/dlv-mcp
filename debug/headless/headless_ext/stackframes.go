package headless_ext

import (
	"fmt"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless"
)

// Stacktrace returns the current goroutine's stack trace
func Stacktrace(session common.Session) (string, error) {
	stacktraceIn := rpc2.StacktraceIn{
		Id:    -1, // current goroutine
		Depth: 20, // reasonable default depth
		Full:  true,
		Cfg: &api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		},
	}

	stackOut, err := sendHeadlessClientRequest[rpc2.StacktraceOut](session, headless.RPCStacktrace, stacktraceIn)
	if err != nil {
		return "", fmt.Errorf("failed to get stacktrace: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("Stack trace:\n")

	for i, frame := range stackOut.Locations {
		// Get the function name, or use "unknown" if not available
		funcName := "unknown"
		if frame.Function != nil {
			funcName = frame.Function.Name()
		}
		builder.WriteString(fmt.Sprintf("%d: %s:%d %s\n", i, frame.File, frame.Line, funcName))
	}

	return builder.String(), nil
}

// SwitchGoroutine switches to a different goroutine
func SwitchGoroutine(session common.Session, goroutineID int) (string, error) {
	// First verify the goroutine exists
	stateIn := rpc2.StateIn{}
	_, err := sendHeadlessClientRequest[rpc2.StateOut](session, headless.RPCState, stateIn)
	if err != nil {
		return "", fmt.Errorf("failed to get state: %w", err)
	}

	// Use the Command RPC to switch goroutine
	_, err = sendHeadlessClientRequest[any](session, headless.RPCCommand, struct {
		Name string
		Expr string
	}{
		Name: "switchGoroutine",
		Expr: fmt.Sprintf("%d", goroutineID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to switch goroutine: %w", err)
	}

	return fmt.Sprintf("Switched to goroutine %d", goroutineID), nil
}

// SwitchThread switches to a different thread
func SwitchThread(session common.Session, threadID int) (string, error) {
	// First verify the thread exists
	stateIn := rpc2.StateIn{}
	_, err := sendHeadlessClientRequest[rpc2.StateOut](session, headless.RPCState, stateIn)
	if err != nil {
		return "", fmt.Errorf("failed to get state: %w", err)
	}

	// Use the Command RPC to switch thread
	_, err = sendHeadlessClientRequest[any](session, headless.RPCCommand, struct {
		Name string
		Expr string
	}{
		Name: "thread",
		Expr: fmt.Sprintf("%d", threadID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to switch thread: %w", err)
	}

	return fmt.Sprintf("Switched to thread %d", threadID), nil
}

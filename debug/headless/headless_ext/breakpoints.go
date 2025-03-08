// Package headless_ext provides extensions to the headless debugger
package headless_ext

import (
	"fmt"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless"
)

// ListBreakpoints lists all breakpoints in the current debug session
func ListBreakpoints(session common.Session) (string, error) {
	// Send ListBreakpoints request
	listBpOut, err := sendHeadlessClientRequest[rpc2.ListBreakpointsOut](session, headless.RPCListBreakpoints, rpc2.ListBreakpointsIn{})
	if err != nil {
		return "", fmt.Errorf("failed to list breakpoints: %w", err)
	}

	// Format the output
	var builder strings.Builder
	builder.WriteString("Breakpoints:\n")

	if len(listBpOut.Breakpoints) == 0 {
		builder.WriteString("No breakpoints set.")
		return builder.String(), nil
	}

	for _, bp := range listBpOut.Breakpoints {
		status := "enabled"
		if bp.Disabled {
			status = "disabled"
		}
		builder.WriteString(fmt.Sprintf("%d: %s:%d (%s)\n", bp.ID, bp.File, bp.Line, status))
	}

	return builder.String(), nil
}

// ToggleBreakpoint enables or disables a breakpoint
func ToggleBreakpoint(session common.Session, breakpointID int) (string, error) {
	// First get the current state of the breakpoint
	listBpOut, err := sendHeadlessClientRequest[rpc2.ListBreakpointsOut](session, headless.RPCListBreakpoints, rpc2.ListBreakpointsIn{})
	if err != nil {
		return "", fmt.Errorf("failed to list breakpoints: %w", err)
	}

	// Find the breakpoint
	var targetBP *api.Breakpoint
	for _, bp := range listBpOut.Breakpoints {
		if bp.ID == breakpointID {
			// Create a copy of the breakpoint to modify
			bpCopy := *bp
			targetBP = &bpCopy
			break
		}
	}

	if targetBP == nil {
		return "", fmt.Errorf("breakpoint %d not found", breakpointID)
	}

	// Toggle the breakpoint
	targetBP.Disabled = !targetBP.Disabled
	newStatus := targetBP.Disabled

	// Create amendment request
	amendBpIn := rpc2.AmendBreakpointIn{
		Breakpoint: *targetBP,
	}

	// Send the request to toggle the breakpoint
	_, err = sendHeadlessClientRequest[any](session, headless.RPCAmendBreakpoint, amendBpIn)
	if err != nil {
		return "", fmt.Errorf("failed to toggle breakpoint: %w", err)
	}

	statusStr := "enabled"
	if newStatus {
		statusStr = "disabled"
	}

	return fmt.Sprintf("Breakpoint %d toggled (now %s)", breakpointID, statusStr), nil
}

// ClearBreakpoint removes a breakpoint
func ClearBreakpoint(session common.Session, breakpointID int) (string, error) {
	// Create clear breakpoint request
	clearBpIn := rpc2.ClearBreakpointIn{
		Id: breakpointID,
	}

	// Send the request to clear the breakpoint
	_, err := sendHeadlessClientRequest[any](session, headless.RPCClearBreakpoint, clearBpIn)
	if err != nil {
		return "", fmt.Errorf("failed to clear breakpoint: %w", err)
	}

	return fmt.Sprintf("Breakpoint %d cleared", breakpointID), nil
}

// CreateWatchpoint creates a watchpoint on a variable
func CreateWatchpoint(session common.Session, variable, scope string, write, read bool) (string, error) {
	// Create a watchpoint (data breakpoint)
	bp := api.Breakpoint{
		Name:      variable,
		WatchExpr: variable,
		WatchType: determineWatchType(read, write),
	}

	// Set scope if provided by adding it to Variables
	if scope != "" {
		bp.Variables = []string{fmt.Sprintf("%s.%s", scope, variable)}
	} else {
		bp.Variables = []string{variable}
	}

	// Create breakpoint request
	createBpIn := rpc2.CreateBreakpointIn{
		Breakpoint: bp,
	}

	// Send the request to create a watchpoint
	createBpOut, err := sendHeadlessClientRequest[rpc2.CreateBreakpointOut](session, headless.RPCCreateBreakpoint, createBpIn)
	if err != nil {
		return "", fmt.Errorf("failed to create watchpoint: %w", err)
	}

	return fmt.Sprintf("Watchpoint %d created on variable '%s' (scope: %s, write: %t, read: %t)",
		createBpOut.Breakpoint.ID, variable, scope, write, read), nil
}

// determineWatchType returns the appropriate watchpoint type based on read/write flags
func determineWatchType(read, write bool) api.WatchType {
	if read && write {
		return api.WatchRead | api.WatchWrite
	} else if read {
		return api.WatchRead
	}
	return api.WatchWrite
}

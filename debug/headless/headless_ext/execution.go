package headless_ext

import (
	"fmt"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless"
)

// Restart restarts the debugged process
func Restart(session common.Session) (string, error) {
	restartIn := rpc2.RestartIn{
		Position:  "",
		ResetArgs: false,
	}

	_, err := sendHeadlessClientRequest[rpc2.RestartOut](session, headless.RPCRestart, restartIn)
	if err != nil {
		return "", fmt.Errorf("failed to restart process: %w", err)
	}

	return "Process restarted", nil
}

// Detach detaches from the debugged process
func Detach(session common.Session, kill bool) (string, error) {
	detachIn := rpc2.DetachIn{
		Kill: kill,
	}

	_, err := sendHeadlessClientRequest[rpc2.DetachOut](session, headless.RPCDetach, detachIn)
	if err != nil {
		return "", fmt.Errorf("failed to detach: %w", err)
	}

	return fmt.Sprintf("Detached from process (kill: %t)", kill), nil
}

// Disassemble disassembles the program at the current location
func Disassemble(session common.Session, startPC uint64, endPC uint64) (string, error) {
	disassembleIn := rpc2.DisassembleIn{
		Scope: api.EvalScope{
			GoroutineID: -1,
			Frame:       0,
		},
		StartPC: startPC,
		EndPC:   endPC,
	}

	disassembleOut, err := sendHeadlessClientRequest[rpc2.DisassembleOut](session, headless.RPCDisassemble, disassembleIn)
	if err != nil {
		return "", fmt.Errorf("failed to disassemble: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("Disassembly:\n")

	if len(disassembleOut.Disassemble) == 0 {
		builder.WriteString("No instructions found.")
		return builder.String(), nil
	}

	for _, instr := range disassembleOut.Disassemble {
		builder.WriteString(fmt.Sprintf("0x%x: %s\n", instr.Loc.PC, instr.Text))
	}

	return builder.String(), nil
}

package headless_ext

import (
	"fmt"
	"strings"

	"github.com/go-delve/delve/service/rpc2"
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless"
)

// CreateCheckpoint creates a checkpoint at the current program state
func CreateCheckpoint(session common.Session) (string, error) {
	checkpointIn := rpc2.CheckpointIn{}

	checkpointOut, err := sendHeadlessClientRequest[rpc2.CheckpointOut](session, headless.RPCCheckpoint, checkpointIn)
	if err != nil {
		return "", fmt.Errorf("failed to create checkpoint: %w", err)
	}

	return fmt.Sprintf("Created checkpoint %d", checkpointOut.ID), nil
}

// ListCheckpoints returns a list of all checkpoints
func ListCheckpoints(session common.Session) (string, error) {
	listCheckpointsIn := rpc2.ListCheckpointsIn{}

	listCheckpointsOut, err := sendHeadlessClientRequest[rpc2.ListCheckpointsOut](session, headless.RPCListCheckpoints, listCheckpointsIn)
	if err != nil {
		return "", fmt.Errorf("failed to list checkpoints: %w", err)
	}

	var builder strings.Builder
	builder.WriteString("Checkpoints:\n")

	if len(listCheckpointsOut.Checkpoints) == 0 {
		builder.WriteString("No checkpoints set.")
		return builder.String(), nil
	}

	for _, cp := range listCheckpointsOut.Checkpoints {
		builder.WriteString(fmt.Sprintf("%d: %s\n", cp.ID, cp.When))
	}

	return builder.String(), nil
}

// ClearCheckpoint removes a checkpoint
func ClearCheckpoint(session common.Session, id int) (string, error) {
	clearCheckpointIn := rpc2.ClearCheckpointIn{
		ID: id,
	}

	_, err := sendHeadlessClientRequest[any](session, headless.RPCClearCheckpoint, clearCheckpointIn)
	if err != nil {
		return "", fmt.Errorf("failed to clear checkpoint: %w", err)
	}

	return fmt.Sprintf("Cleared checkpoint %d", id), nil
}

package headless_ext

import (
	"fmt"
	"strings"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless"
)

// ListLocalVars returns a list of local variables in the current scope
// Uses the RPCServer.ListLocalVars API method:
// https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ListLocalVars
func ListLocalVars(session common.Session) (string, error) {
	// Create the request using the official API type
	listVarsIn := rpc2.ListLocalVarsIn{
		Scope: api.EvalScope{
			GoroutineID: -1,
			Frame:       0,
		},
		Cfg: api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		},
	}

	// Call the ListLocalVars RPC method
	listVarsOut, err := sendHeadlessClientRequest[rpc2.ListLocalVarsOut](
		session,
		headless.RPCListLocalVars,
		listVarsIn,
	)
	if err != nil {
		return "", fmt.Errorf("failed to list local variables: %w", err)
	}

	// Format the response
	var builder strings.Builder
	builder.WriteString("Local variables:\n")

	if len(listVarsOut.Variables) == 0 {
		builder.WriteString("No local variables found.")
		return builder.String(), nil
	}

	for _, v := range listVarsOut.Variables {
		formatVariable(&builder, &v, 0)
	}
	return builder.String(), nil
}

// ListFunctionArgs returns a list of function arguments in the current scope
// Uses the RPCServer.ListFunctionArgs API method:
// https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ListFunctionArgs
func ListFunctionArgs(session common.Session) (string, error) {
	// Create the request using the official API type
	listArgsIn := rpc2.ListFunctionArgsIn{
		Scope: api.EvalScope{
			GoroutineID: -1,
			Frame:       0,
		},
		Cfg: api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		},
	}

	// Call the ListFunctionArgs RPC method
	listArgsOut, err := sendHeadlessClientRequest[rpc2.ListFunctionArgsOut](
		session,
		headless.RPCListFunctionArgs,
		listArgsIn,
	)
	if err != nil {
		return "", fmt.Errorf("failed to list function arguments: %w", err)
	}

	// Format the response
	var builder strings.Builder
	builder.WriteString("Function arguments:\n")

	if len(listArgsOut.Args) == 0 {
		builder.WriteString("No function arguments found.")
		return builder.String(), nil
	}

	for _, v := range listArgsOut.Args {
		formatVariable(&builder, &v, 0)
	}
	return builder.String(), nil
}

// SetVariable sets the value of a variable
// Uses the RPCServer.Set API method:
// https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.Set
func SetVariable(session common.Session, name, value string) (string, error) {
	// Create the request using the official API type
	setIn := rpc2.SetIn{
		Scope: api.EvalScope{
			GoroutineID: -1,
			Frame:       0,
		},
		Symbol: name,
		Value:  value,
	}

	// Call the Set RPC method
	_, err := sendHeadlessClientRequest[rpc2.SetOut](
		session,
		headless.RPCSet,
		setIn,
	)
	if err != nil {
		return "", fmt.Errorf("failed to set variable: %w", err)
	}

	return fmt.Sprintf("Variable %s set to %s", name, value), nil
}

// ExamineMemory examines memory at the given address
// Uses the RPCServer.ExamineMemory API method:
// https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ExamineMemory
func ExamineMemory(session common.Session, address string, length int) (string, error) {
	// Parse the address to uint64
	var addr uint64
	_, err := fmt.Sscanf(address, "0x%x", &addr)
	if err != nil {
		// Try without 0x prefix
		_, err = fmt.Sscanf(address, "%x", &addr)
		if err != nil {
			return "", fmt.Errorf("failed to parse address '%s': %w", address, err)
		}
	}

	// Create the request using the official API type
	examineIn := rpc2.ExamineMemoryIn{
		Address: addr,
		Length:  length,
	}

	// Call the ExamineMemory RPC method
	examineOut, err := sendHeadlessClientRequest[rpc2.ExaminedMemoryOut](
		session,
		headless.RPCExamineMemory,
		examineIn,
	)
	if err != nil {
		return "", fmt.Errorf("failed to examine memory: %w", err)
	}

	// Format the response
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Memory at %s:\n", address))

	if len(examineOut.Mem) == 0 {
		builder.WriteString("No memory data found.")
		return builder.String(), nil
	}

	// Format memory as hex bytes with ASCII representation
	formatMemoryDump(&builder, examineOut.Mem, addr, examineOut.IsLittleEndian)
	return builder.String(), nil
}

// formatMemoryDump formats memory data as a hex dump with ASCII representation
func formatMemoryDump(builder *strings.Builder, mem []byte, baseAddr uint64, isLittleEndian bool) {
	const bytesPerLine = 16

	for i := 0; i < len(mem); i += bytesPerLine {
		// Address
		builder.WriteString(fmt.Sprintf("0x%016x: ", baseAddr+uint64(i)))

		// Hex bytes
		end := i + bytesPerLine
		if end > len(mem) {
			end = len(mem)
		}

		// Print hex values
		for j := i; j < end; j++ {
			builder.WriteString(fmt.Sprintf("%02x ", mem[j]))
		}

		// Padding if line is incomplete
		for j := end; j < i+bytesPerLine; j++ {
			builder.WriteString("   ")
		}

		// ASCII representation
		builder.WriteString(" |")
		for j := i; j < end; j++ {
			// Print printable ASCII characters; replace others with '.'
			if mem[j] >= 32 && mem[j] <= 126 {
				builder.WriteByte(mem[j])
			} else {
				builder.WriteByte('.')
			}
		}
		builder.WriteString("|\n")
	}
}

// formatVariable formats a variable for display
func formatVariable(builder *strings.Builder, v *api.Variable, depth int) {
	indent := strings.Repeat("  ", depth)

	if v.Name != "" {
		builder.WriteString(fmt.Sprintf("%s%s = ", indent, v.Name))
	} else {
		builder.WriteString(indent)
	}

	if v.Type != "" {
		builder.WriteString(fmt.Sprintf("(%s) ", v.Type))
	}

	if len(v.Children) > 0 {
		builder.WriteString("{\n")
		for i := range v.Children {
			formatVariable(builder, &v.Children[i], depth+1)
		}
		builder.WriteString(fmt.Sprintf("%s}\n", indent))
	} else {
		builder.WriteString(fmt.Sprintf("%v\n", v.Value))
	}
}

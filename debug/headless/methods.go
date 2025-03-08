package headless

type RPCMethod string

const (
	RPCCommand          RPCMethod = "RPCServer.Command"
	RPCState            RPCMethod = "RPCServer.State"
	RPCCreateBreakpoint RPCMethod = "RPCServer.CreateBreakpoint"
	RPCEval             RPCMethod = "RPCServer.Eval"

	// Breakpoint methods
	RPCListBreakpoints RPCMethod = "RPCServer.ListBreakpoints"
	RPCClearBreakpoint RPCMethod = "RPCServer.ClearBreakpoint"
	RPCAmendBreakpoint RPCMethod = "RPCServer.AmendBreakpoint"

	// Stack methods
	RPCStacktrace      RPCMethod = "RPCServer.Stacktrace"
	RPCSwitchGoroutine RPCMethod = "RPCServer.SwitchGoroutine"
	RPCSwitchThread    RPCMethod = "RPCServer.SwitchThread"

	// Checkpoint methods
	RPCCheckpoint      RPCMethod = "RPCServer.Checkpoint"
	RPCListCheckpoints RPCMethod = "RPCServer.ListCheckpoints"
	RPCClearCheckpoint RPCMethod = "RPCServer.ClearCheckpoint"

	// Execution methods
	RPCRestart     RPCMethod = "RPCServer.Restart"
	RPCDetach      RPCMethod = "RPCServer.Detach"
	RPCDisassemble RPCMethod = "RPCServer.Disassemble"

	// Variable methods
	// Documentation: https://pkg.go.dev/github.com/go-delve/delve/service/rpc2
	RPCListLocalVars    RPCMethod = "RPCServer.ListLocalVars"    // https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ListLocalVars
	RPCListFunctionArgs RPCMethod = "RPCServer.ListFunctionArgs" // https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ListFunctionArgs
	RPCSet              RPCMethod = "RPCServer.Set"              // https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.Set
	RPCExamineMemory    RPCMethod = "RPCServer.ExamineMemory"    // https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ExamineMemory

	// Source code methods
	RPCListSources RPCMethod = "RPCServer.ListSources" // https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ListSources
)

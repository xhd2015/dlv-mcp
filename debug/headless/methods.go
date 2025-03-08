package headless

type RPCMethod string

const (
	RPCCommand          RPCMethod = "RPCServer.Command"
	RPCState            RPCMethod = "RPCServer.State"
	RPCCreateBreakpoint RPCMethod = "RPCServer.CreateBreakpoint"
	RPCEval             RPCMethod = "RPCServer.Eval"
)

package headless_ext

import (
	"fmt"

	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless"
)

func sendHeadlessClientRequest[T any](session common.Session, method headless.RPCMethod, params interface{}) (T, error) {
	var result T
	// Cast to headless.Session to access internals
	headlessSession, ok := session.(*headless.Session)
	if !ok {
		return result, fmt.Errorf("session is not a headless session")
	}
	client := headlessSession.Client
	if client == nil {
		return result, fmt.Errorf("client is nil")
	}

	return headless.SendHeadlessClientRequest[T](client, headless.RPCMethod(method), params)
}

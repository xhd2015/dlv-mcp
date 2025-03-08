package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/xhd2015/debugger-mcp/tools/debug"
)

// install: go install ./cmd/dlv-mcp
const help = `
dlv-mcp dlv mcp server

Usage: dlv-mcp <cmd> [OPTIONS]

Available commands:
  help                               show help message

Options:
  --debugger <debugger>              Type of debugger to use: 'headless'(default) or 'dap'
  --help   show help message
`

func main() {
	if err := handle(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handle(args []string) error {
	if len(args) > 0 && args[0] == "help" {
		fmt.Println(strings.TrimSpace(help))
		return nil
	}

	var debugger string
	n := len(args)
	for i, arg := range args {
		switch arg {
		case "--debugger":
			if i+1 >= n {
				return fmt.Errorf("%s requires arg", arg)
			}
			debugger = args[i+1]
		case "-h", "--help":
			fmt.Println(strings.TrimSpace(help))
			return nil
		}
	}

	if debugger == "" {
		debugger = "headless"
	}

	// Create MCP server
	s := server.NewMCPServer(
		"Go Delve Debugger MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithResourceCapabilities(true, true),
	)

	// Register tools
	if err := debug.RegisterTools(s, debug.ToolOptions{
		DebuggerType: debugger,
	}); err != nil {
		return err
	}

	// Start the server with our monitored context
	log.Printf("Starting MCP server with custom context...")
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
	return nil
}

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/dlv-mcp/tools/debug"
	"github.com/xhd2015/dlv-mcp/vendir/third-party/github.com/mark3labs/mcp-go/server"
)

// install: go install ./cmd/dlv-mcp
const help = `
dlv-mcp dlv mcp server

Usage: dlv-mcp <cmd> [OPTIONS]

Available commands:
  help                               show help message

Options:
  --debugger <debugger>              Type of debugger to use: 'headless'(default) or 'dap'
  --listen <listen>                   Listen address (default: 127.0.0.1:12763)
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

	var listen string
	var debugger string
	n := len(args)
	for i, arg := range args {
		switch arg {
		case "--debugger":
			if i+1 >= n {
				return fmt.Errorf("%s requires arg", arg)
			}
			debugger = args[i+1]
		case "--listen":
			if i+1 >= n {
				return fmt.Errorf("%s requires arg", arg)
			}
			listen = args[i+1]
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

	// for append log to file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}
	configDir := filepath.Join(homeDir, ".dlv-mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}
	logFile := filepath.Join(configDir, "dlv-mcp.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	logger := &logger{
		writer: file,
	}

	// Register tools
	if err := debug.RegisterTools(s, debug.ToolOptions{
		DebuggerType: debugger,
		Logger:       logger,
	}); err != nil {
		return err
	}

	// Start the server with our monitored context

	if listen == "" {
		log.Printf("MCP Server listening on stdio...")
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Printf("MCP Server listening on %s...", listen)
		sseServer := server.NewSSEServer(s)
		if err := sseServer.Start(listen); err != nil {
			return err
		}
	}
	return nil
}

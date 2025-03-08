package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// MCP Request types
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	ID      string      `json:"id"`
	Params  interface{} `json:"params"`
}

type CallToolParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// Content types
type ContentType string

const (
	ContentTypeText ContentType = "text"
)

type Content struct {
	Type ContentType `json:"type"`
	Text string      `json:"text"`
}

// MCP Response types
type MCPResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      string `json:"id"`
	Result  struct {
		Content []Content `json:"content"`
		IsError bool      `json:"isError,omitempty"`
	} `json:"result,omitempty"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func main() {
	// Add command-line flags for debug mode
	debugMode := flag.String("mode", "binary", "Debug mode: 'binary' for debugging binaries, 'source' for debugging source directly")
	debuggerType := flag.String("debugger", "headless", "Debugger type: 'dap' for DAP protocol, 'headless' for Delve headless mode (default: headless)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Demonstrates the Delve MCP debugger integration.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -mode=binary|source        Debug mode (default: binary)\n")
		fmt.Fprintf(os.Stderr, "      binary: Debug compiled binary\n")
		fmt.Fprintf(os.Stderr, "      source: Debug source code directly\n\n")
		fmt.Fprintf(os.Stderr, "  -debugger=dap|headless    Debugger type (default: headless)\n")
		fmt.Fprintf(os.Stderr, "      dap: Use DAP protocol (better compatibility)\n")
		fmt.Fprintf(os.Stderr, "      headless: Use Delve headless mode (better for variable evaluation)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s                                # Run in binary mode with headless debugger\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -mode=source                   # Run in source mode with headless debugger\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -debugger=dap                  # Run in binary mode with DAP debugger\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -mode=source -debugger=dap     # Run in source mode with DAP debugger\n", os.Args[0])
	}
	flag.Parse()

	// Validate debug mode
	if *debugMode != "binary" && *debugMode != "source" {
		log.Fatalf("Invalid debug mode: %s. Must be 'binary' or 'source'", *debugMode)
	}

	// Validate debugger type
	if *debuggerType != "dap" && *debuggerType != "headless" {
		log.Fatalf("Invalid debugger type: %s. Must be 'dap' or 'headless'", *debuggerType)
	}

	log.Printf("=== Delve MCP Server Demo (Mode: %s, Debugger: %s) ===", *debugMode, *debuggerType)

	// Get absolute path to the testdata/hello.go file
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Use hello.go instead of main.go since that's what exists in testdata
	sourceFile := filepath.Join(workingDir, "cmd", "dlv-mcp", "testdata", "hello.go")
	testProgram := filepath.Join(workingDir, "cmd", "dlv-mcp", "testdata", "__debug_bin")

	// Check if source file exists
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		log.Fatalf("Source file %s does not exist", sourceFile)
	}

	// Only build the binary if we're in binary mode
	if *debugMode == "binary" {
		// Builder
		builder := exec.Command("go", "build", "-gcflags=all=-N -l", "-o", testProgram, "./cmd/dlv-mcp/testdata")
		builder.Dir = workingDir
		builder.Stdout = os.Stdout
		builder.Stderr = os.Stderr
		if err := builder.Run(); err != nil {
			log.Fatalf("Failed to build debug binary: %v", err)
		}
	}

	// Get absolute paths
	absSourceFile, err := filepath.Abs(sourceFile)
	if err != nil {
		log.Fatalf("Failed to get absolute path for source file: %v", err)
	}

	var absTestProgram string
	if *debugMode == "binary" {
		absTestProgram, err = filepath.Abs(testProgram)
		if err != nil {
			log.Fatalf("Failed to get absolute path for test program: %v", err)
		}
	}

	// Kill any existing dlv processes that might be using the port
	killCmd := exec.Command("pkill", "-f", "dlv")
	killCmd.Run() // Ignore errors, as there might not be any processes to kill

	// Start the MCP server as a child process
	serverCmd := exec.Command("go", "run", "./cmd/dlv-mcp", "-debugger="+*debuggerType)
	stdin, err := serverCmd.StdinPipe()
	if err != nil {
		log.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := serverCmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := serverCmd.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create stderr pipe: %v", err)
	}

	// Forward stderr to our stderr for debugging
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "SERVER INFO: %s\n", scanner.Text())
		}
	}()

	log.Println("Starting MCP server...")
	if err := serverCmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Give the server a moment to start up
	time.Sleep(1 * time.Second)

	// Create a buffered reader for the server's stdout
	reader := bufio.NewReader(stdout)

	// Define a function to send a command and get a response
	sendCommand := func(name string, args map[string]interface{}) (*MCPResponse, error) {
		// Create an MCP request
		req := MCPRequest{
			JSONRPC: "2.0",
			Method:  "tools/call",
			ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
			Params: CallToolParams{
				Name:      name,
				Arguments: args,
			},
		}

		// Marshal the request to JSON
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %v", err)
		}

		log.Printf("Sending command: %s", name)
		log.Printf("Request: %s", string(reqJSON))

		// Send the request to the server
		if _, err := fmt.Fprintf(stdin, "%s\n", reqJSON); err != nil {
			return nil, fmt.Errorf("failed to send request: %v", err)
		}

		// Read the response
		respJSON, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("server closed the connection")
			}
			return nil, fmt.Errorf("failed to read response: %v", err)
		}

		log.Printf("Response: %s", respJSON)

		// Check if the response is valid JSON
		respJSON = strings.TrimSpace(respJSON)
		if !strings.HasPrefix(respJSON, "{") {
			return nil, fmt.Errorf("server returned non-JSON response: %s", respJSON)
		}

		// Unmarshal the response
		var resp MCPResponse
		if err := json.Unmarshal([]byte(respJSON), &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response: %v", err)
		}

		// Check for protocol-level errors
		if resp.Error != nil {
			return nil, fmt.Errorf("server error: %s", resp.Error.Message)
		}

		// Check for tool-level errors
		if resp.Result.IsError && len(resp.Result.Content) > 0 {
			return nil, fmt.Errorf("tool error: %s", resp.Result.Content[0].Text)
		}

		return &resp, nil
	}

	// Helper function to get response content text
	getResponseText := func(resp *MCPResponse) string {
		if len(resp.Result.Content) > 0 {
			return resp.Result.Content[0].Text
		}
		return ""
	}

	// Demo a full debugging workflow

	// 1. Start a debug session
	log.Println("\n=== Starting debug session ===")

	debugArgs := map[string]interface{}{
		"args": []string{},
	}

	// Set the program or source based on debug mode
	if *debugMode == "binary" {
		debugArgs["program"] = absTestProgram
		debugArgs["mode"] = "exec"
	} else {
		// For source debugging, we need to use a different mode
		debugArgs["program"] = absSourceFile
		debugArgs["mode"] = "debug" // Use "debug" mode for source files
	}

	resp, err := sendCommand("start_debug", debugArgs)
	if err != nil {
		log.Fatalf("Failed to start debug session: %v", err)
	}

	// Extract the session ID from the response content
	content := getResponseText(resp)

	sessionID := ""
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Debug session started with ID: ") {
			sessionID = strings.TrimPrefix(line, "Debug session started with ID: ")
			break
		}
	}

	if sessionID == "" {
		log.Fatalf("Failed to extract session ID from response: %s", content)
	}

	log.Printf("Session ID: %s", sessionID)

	// Wait for a moment to make sure the debug session is fully initialized
	// Increase wait time to 3 seconds to give Delve more time to load debug symbols
	log.Println("Waiting for debug session to fully initialize...")
	time.Sleep(3 * time.Second)

	// 2. Set a breakpoint at the 'add' function (line 21)
	log.Println("\n=== Setting breakpoint ===")
	_, err = sendCommand("set_breakpoint", map[string]interface{}{
		"session_id": sessionID,
		"file":       absSourceFile,
		"line":       21,
	})
	if err != nil {
		log.Fatalf("Failed to set breakpoint: %v", err)
	}
	log.Println("Breakpoint set successfully")

	// Wait for a moment to make sure the breakpoint is properly registered
	time.Sleep(1 * time.Second)

	// 3. Continue execution to hit the breakpoint
	log.Println("\n=== Continuing to breakpoint ===")
	_, err = sendCommand("continue", map[string]interface{}{
		"session_id": sessionID,
	})
	if err != nil {
		log.Printf("Warning: Failed to continue execution: %v", err)
		// Continue with the demo even if the continue command fails
	} else {
		log.Println("Program is running")

		// Wait for a moment to let the program hit the breakpoint
		time.Sleep(2 * time.Second)

		// 4. Evaluate variables at the breakpoint
		log.Println("\n=== Evaluating variables ===")
		aResp, err := sendCommand("evaluate", map[string]interface{}{
			"session_id": sessionID,
			"expression": "a",
		})
		if err != nil {
			log.Printf("Warning: Failed to evaluate 'a': %v", err)
		} else {
			log.Printf("a = %s", getResponseText(aResp))
		}

		bResp, err := sendCommand("evaluate", map[string]interface{}{
			"session_id": sessionID,
			"expression": "b",
		})
		if err != nil {
			log.Printf("Warning: Failed to evaluate 'b': %v", err)
		} else {
			log.Printf("b = %s", getResponseText(bResp))
		}

		// 5. Step over the current line
		log.Println("\n=== Stepping over line ===")
		_, err = sendCommand("next", map[string]interface{}{
			"session_id": sessionID,
		})
		if err != nil {
			log.Printf("Warning: Failed to step over: %v", err)
		} else {
			log.Println("Stepped over line")
		}

		// 6. Continue to program completion
		log.Println("\n=== Continuing to completion ===")
		_, err = sendCommand("continue", map[string]interface{}{
			"session_id": sessionID,
		})
		if err != nil {
			log.Printf("Warning: Failed to continue execution: %v", err)
		} else {
			log.Println("Program is running to completion")

			// Wait for the program to complete
			time.Sleep(1 * time.Second)

			// Note: The program will likely terminate on its own after this point
			log.Println("Program execution completed")
		}
	}

	// Wait for the program to complete
	time.Sleep(1 * time.Second)

	// 7. Terminate the debug session
	log.Println("\n=== Terminating debug session ===")
	termResp, err := sendCommand("terminate_debug", map[string]interface{}{
		"session_id": sessionID,
	})
	if err != nil {
		log.Printf("Warning: Failed to terminate debug session: %v", err)
	} else {
		log.Printf("Debug session terminated: %s", getResponseText(termResp))
	}

	// Clean up
	log.Println("\n=== Stopping MCP server ===")
	if err := gracefulShutdown(serverCmd, 3*time.Second); err != nil {
		log.Printf("Failed to gracefully shutdown server: %v", err)
	}
	log.Println("=== Demo completed successfully ===")
}

func gracefulShutdown(serverCmd *exec.Cmd, timeout time.Duration) error {
	// First, try sending an interrupt signal
	if err := serverCmd.Process.Signal(os.Interrupt); err != nil {
		log.Printf("Failed to send interrupt signal: %v", err)
	}

	// Set a timeout for waiting
	done := make(chan error)
	go func() {
		done <- serverCmd.Wait()
	}()

	// Wait for the server to exit with a timeout
	select {
	case err := <-done:
		if err != nil {
			log.Printf("Server exited with error: %v", err)
		} else {
			log.Printf("Server exited cleanly")
		}
	case <-time.After(timeout):
		// If it doesn't exit within the timeout, force kill it
		log.Println("Server didn't exit cleanly, killing process...")

		// On Unix systems, SIGKILL is more reliable than Process.Kill()
		if err := serverCmd.Process.Signal(syscall.SIGKILL); err != nil {
			log.Printf("Failed to kill server process: %v", err)
		} else {
			log.Printf("Sent SIGKILL to server process")
		}

		// Wait for the process to be killed
		<-done
	}

	return nil
}

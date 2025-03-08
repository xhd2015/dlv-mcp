package headless

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
	"github.com/google/uuid"
	"github.com/xhd2015/dlv-mcp/debug/common"
)

// SessionManager manages headless debug sessions
type SessionManager struct {
	debuggerType string
	sessions     map[string]common.Session
	mu           sync.Mutex
}

// NewSessionManager creates a new headless session manager
func NewSessionManager() common.SessionManager {
	return &SessionManager{
		debuggerType: "headless",
		sessions:     make(map[string]common.Session),
	}
}

// GetDebuggerType returns the type of debugger being used
func (sm *SessionManager) GetDebuggerType() string {
	return sm.debuggerType
}

// NewSession creates a new headless debug session
func (sm *SessionManager) NewSession(programPath string, args []string, mode string) (common.Session, error) {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Creating session for program: %s, mode: %s\n", programPath, mode)

	// Generate a session ID
	sessionID := fmt.Sprintf("session-%d", uuid.New().ID())

	var dlvCmd *exec.Cmd
	var client *Client
	var err error

	if mode == "remote" {
		// For remote mode, we don't start a server
		client = NewClient()
		// The actual connection will be established by the tool
	} else {
		// Determine the correct command based on mode
		dlvCommand := "debug"
		if mode == "exec" {
			dlvCommand = "exec"
		} else if mode == "test" {
			dlvCommand = "test"
		}

		// Start the Delve headless server
		port := "54321" // Hardcoded for simplicity
		fmt.Fprintf(os.Stderr, "DEBUG Session: Starting Delve in headless mode\n")

		// For headless mode, we need to specify the command (debug, exec, test)
		// and use the --headless flag
		dlvCmd = exec.Command("dlv", dlvCommand, "--headless", "--api-version=2", "--listen=127.0.0.1:"+port, programPath)

		if err := dlvCmd.Start(); err != nil {
			return nil, fmt.Errorf("failed to start Delve headless server: %w", err)
		}

		// Give the server a moment to start up
		time.Sleep(1 * time.Second)

		// Connect to the headless server
		client = NewClient()
		err = client.Connect(context.Background(), "127.0.0.1:"+port)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to headless server: %w", err)
		}

		// Initialize the debug session
		err = client.Initialize(programPath, args, mode)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize debug session: %w", err)
		}
	}

	// Create a new session
	session := &Session{
		id:       sessionID,
		Client:   client,
		program:  programPath,
		cmd:      dlvCmd,
		isPaused: false,
	}

	// Store session
	sm.mu.Lock()
	sm.sessions[sessionID] = session
	sm.mu.Unlock()

	return session, nil
}

// CreateSession creates a new debug session with the given parameters
func (sm *SessionManager) CreateSession(ctx context.Context, programPath string, args []string, mode string) (*common.SessionInfo, error) {
	session, err := sm.NewSession(programPath, args, mode)
	if err != nil {
		return nil, err
	}

	// Cast to our Session type to set working directory
	if s, ok := session.(*Session); ok {
		// For remote sessions, working directory will be set by the tool
		s.workingDir = filepath.Dir(programPath)
	}

	// Return session info
	return &common.SessionInfo{
		ID:          session.GetID(),
		ProgramPath: programPath,
		State:       "created",
		WorkingDir:  filepath.Dir(programPath),
	}, nil
}

// TerminateSession terminates a debug session
func (sm *SessionManager) TerminateSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Terminate the session
	err := session.Terminate()
	if err != nil {
		return err
	}

	// Remove from sessions map
	delete(sm.sessions, sessionID)

	return nil
}

// ListSessions returns a list of active debug sessions
func (sm *SessionManager) ListSessions() []*common.SessionInfo {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var result []*common.SessionInfo
	for id, session := range sm.sessions {
		s := session.(*Session) // Type assertion
		state := "running"
		if s.isPaused {
			state = "paused"
		}

		result = append(result, &common.SessionInfo{
			ID:          id,
			ProgramPath: s.program,
			State:       state,
			WorkingDir:  s.workingDir,
		})
	}

	return result
}

// GetSession returns a debug session by ID
func (sm *SessionManager) GetSession(sessionID string) (common.Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, ok := sm.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// Session represents a headless debug session
type Session struct {
	id         string
	Client     *Client
	program    string
	cmd        *exec.Cmd
	isPaused   bool
	workingDir string
}

// SetWorkingDir sets the working directory for the session
func (s *Session) SetWorkingDir(dir string) {
	s.workingDir = dir
}

// GetWorkingDir returns the working directory for the session
func (s *Session) GetWorkingDir() string {
	return s.workingDir
}

// GetID returns the session ID
func (s *Session) GetID() string {
	return s.id
}

// SetBreakpoint sets a breakpoint at the given file and line
func (s *Session) SetBreakpoint(file string, line int) (int, error) {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Setting breakpoint at %s:%d\n", file, line)

	// Create a structured breakpoint request using the proper type
	bp := api.Breakpoint{
		File: file,
		Line: line,
	}

	// Use the proper typed request structure from Delve
	createBpIn := rpc2.CreateBreakpointIn{
		Breakpoint: bp,
	}

	// Send the request to the Delve with a typed response
	response, err := SendHeadlessClientRequest[rpc2.CreateBreakpointOut](s.Client, RPCCreateBreakpoint, createBpIn)
	if err != nil {
		return 0, fmt.Errorf("failed to set breakpoint: %w", err)
	}

	// Log the successful response
	fmt.Fprintf(os.Stderr, "DEBUG Session: Breakpoint created successfully\n")

	// Return the breakpoint ID directly from the typed response
	return response.Breakpoint.ID, nil
}

// Continue continues execution until the next breakpoint
func (s *Session) Continue() error {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Continuing execution\n")

	// Create a command using the proper Delve type
	// The command is a special case that doesn't have an In structure
	// but we can directly use a map with the command name
	cmdRequest := api.DebuggerCommand{
		Name: "continue",
	}

	// Send the request to the Delve server with a typed response
	response, err := SendHeadlessClientRequest[rpc2.CommandOut](s.Client, RPCCommand, cmdRequest)
	if err != nil {
		return fmt.Errorf("failed to continue execution: %w", err)
	}

	// Update the state directly from the typed response
	s.isPaused = response.State.NextInProgress
	fmt.Fprintf(os.Stderr, "DEBUG Session: Paused state after continue: %v\n", s.isPaused)

	fmt.Fprintf(os.Stderr, "DEBUG Session: Continue command sent successfully\n")
	return nil
}

// Next steps over the current line
func (s *Session) Next() error {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Stepping over line\n")

	// Create a command using the proper structure
	cmdRequest := api.DebuggerCommand{
		Name: "next",
	}

	// Send next request with a typed response
	response, err := SendHeadlessClientRequest[rpc2.CommandOut](s.Client, RPCCommand, cmdRequest)
	if err != nil {
		return fmt.Errorf("failed to step over line: %w", err)
	}

	// Log the result
	fmt.Fprintf(os.Stderr, "DEBUG Session: Next command sent successfully\n")

	// Update the state directly from the typed response
	s.isPaused = response.State.NextInProgress
	fmt.Fprintf(os.Stderr, "DEBUG Session: After next, program paused state: %v\n", s.isPaused)
	return nil
}

// StepIn steps into the current function
func (s *Session) StepIn() error {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Stepping into function\n")

	// Create a command request with the proper command name
	cmdRequest := api.DebuggerCommand{
		Name: "step",
	}

	// Send the request to the Delve server with a typed response
	response, err := SendHeadlessClientRequest[rpc2.CommandOut](s.Client, RPCCommand, cmdRequest)
	if err != nil {
		return fmt.Errorf("failed to step into function: %w", err)
	}

	// Update state directly from typed response
	s.isPaused = response.State.NextInProgress
	fmt.Fprintf(os.Stderr, "DEBUG Session: After step in, program paused state: %v\n", s.isPaused)

	fmt.Fprintf(os.Stderr, "DEBUG Session: Step in command sent successfully\n")
	return nil
}

// StepOut steps out of the current function
func (s *Session) StepOut() error {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Stepping out of current function\n")

	// Create a command request with the proper command name
	cmdRequest := api.DebuggerCommand{
		Name: "stepOut",
	}

	// Send the request to the Delve server with a typed response
	response, err := SendHeadlessClientRequest[rpc2.CommandOut](s.Client, RPCCommand, cmdRequest)
	if err != nil {
		return fmt.Errorf("failed to step out of function: %w", err)
	}

	// Update state directly from typed response
	s.isPaused = response.State.NextInProgress
	fmt.Fprintf(os.Stderr, "DEBUG Session: After step out, program paused state: %v\n", s.isPaused)

	fmt.Fprintf(os.Stderr, "DEBUG Session: Step out command sent successfully\n")
	return nil
}

// Evaluate evaluates an expression in the current context
func (s *Session) Evaluate(expr string) (string, error) {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Evaluating expression: %s\n", expr)

	// Create a properly typed eval request
	evalIn := rpc2.EvalIn{
		Scope: api.EvalScope{
			GoroutineID: -1, // Use current goroutine
			Frame:       0,  // Top frame
		},
		Expr: expr,
		Cfg: &api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		},
	}

	// Send the request to the Delve server with a typed response
	response, err := SendHeadlessClientRequest[rpc2.EvalOut](s.Client, RPCEval, evalIn)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression: %w", err)
	}

	// Log the response for debugging
	fmt.Fprintf(os.Stderr, "DEBUG Session: Received evaluation response\n")

	// Format the variable from the typed response
	if response.Variable != nil {
		result := formatVariableTyped(response.Variable, 0)
		return result, nil
	}

	return "", nil
}

// formatVariableTyped formats a variable with the proper api.Variable type
func formatVariableTyped(variable *api.Variable, depth int) string {
	if variable == nil {
		return "<nil>"
	}

	if depth > 3 {
		return fmt.Sprintf("%s (truncated)", variable.Type)
	}

	// Handle different variable types
	switch variable.Kind {
	case reflect.String:
		return fmt.Sprintf("%q", variable.Value)
	case reflect.Slice, reflect.Array:
		if variable.Len == 0 {
			return "[]"
		}
		return fmt.Sprintf("%s (len=%d)", variable.Type, variable.Len)
	case reflect.Ptr:
		if len(variable.Children) == 0 {
			if variable.Type == "" {
				return "nil"
			}
			return fmt.Sprintf("(%s) nil", variable.Type)
		}
		return formatVariableTyped(&variable.Children[0], depth+1)
	case reflect.Struct:
		result := fmt.Sprintf("%s {", variable.Type)
		if len(variable.Children) > 0 {
			result += " ... "
		}
		result += "}"
		return result
	case reflect.Map:
		return fmt.Sprintf("map[%s] (len=%d)", variable.Type, variable.Len)
	case reflect.Interface:
		if len(variable.Children) > 0 {
			return formatVariableTyped(&variable.Children[0], depth+1)
		}
		return fmt.Sprintf("(%s) nil", variable.Type)
	default:
		return variable.Value
	}
}

// Terminate terminates the debug session
func (s *Session) Terminate() error {
	// First, check if the program is still running by getting its state
	if !s.isExited() {
		// If the program is still running, send the exit command
		fmt.Fprintf(os.Stderr, "DEBUG Session: Sending exit command to terminate debugging\n")
		_, err := SendHeadlessClientRequest[rpc2.CommandOut](s.Client, RPCCommand, map[string]interface{}{
			"name": "exit",
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG Session: Warning: Error sending exit command: %v\n", err)
			// Continue with the cleanup even if the exit command fails
		}
	} else {
		fmt.Fprintf(os.Stderr, "DEBUG Session: Program has already exited, skipping exit command\n")
	}

	// Close the client connection
	fmt.Fprintf(os.Stderr, "DEBUG Session: Closing debugger client connection\n")
	if err := s.Client.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to close client connection: %v\n", err)
	}

	// Kill the Delve process
	if s.cmd != nil && s.cmd.Process != nil {
		fmt.Fprintf(os.Stderr, "DEBUG Session: Killing Delve process\n")
		if err := s.cmd.Process.Kill(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to kill Delve process: %v\n", err)
		}
	}

	return nil
}

// isExited checks if the debug target has exited
func (s *Session) isExited() bool {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Checking if program has exited\n")

	// Create a properly typed state request
	stateIn := rpc2.StateIn{
		NonBlocking: true,
	}

	// Send a request to get the current state with a typed response
	response, err := SendHeadlessClientRequest[rpc2.StateOut](s.Client, RPCState, stateIn)
	if err != nil {
		// Common error patterns that indicate the program has exited
		exitPatterns := []string{
			"process exited",
			"has exited with status",
			"process not found",
			"no such process",
		}

		for _, pattern := range exitPatterns {
			if strings.Contains(err.Error(), pattern) {
				fmt.Fprintf(os.Stderr, "DEBUG Session: Program has exited according to error: %v\n", err)
				return true
			}
		}

		// For other errors, log but assume exited to be safe
		fmt.Fprintf(os.Stderr, "DEBUG Session: Error getting program state, assuming exited: %v\n", err)
		return true
	}

	// Check exited status directly from the typed response
	exited := false
	if response.State != nil && response.State.Exited {
		fmt.Fprintf(os.Stderr, "DEBUG Session: Program has exited with status %d\n", response.State.ExitStatus)
		exited = true
	}

	return exited
}

// IsPaused returns whether the debug session is paused
func (s *Session) IsPaused() bool {
	return s.isPaused
}

// ConnectRemote connects to a remote debugger
func (s *Session) ConnectRemote(ctx context.Context, address string) error {
	if err := s.Client.Connect(ctx, address); err != nil {
		return fmt.Errorf("failed to connect to remote debugger: %w", err)
	}
	return nil
}

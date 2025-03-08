package dap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/xhd2015/debugger-mcp/debug/common"
)

// SessionManager manages DAP debug sessions
type SessionManager struct {
	debuggerType string
	sessions     map[string]common.Session
	mu           sync.Mutex
}

// NewSessionManager creates a new DAP session manager
func NewSessionManager() common.SessionManager {
	return &SessionManager{
		debuggerType: "dap",
		sessions:     make(map[string]common.Session),
	}
}

// GetDebuggerType returns the type of debugger being used
func (sm *SessionManager) GetDebuggerType() string {
	return sm.debuggerType
}

// NewSession creates a new DAP debug session
func (sm *SessionManager) NewSession(programPath string, args []string, mode string) (common.Session, error) {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Creating session for program: %s, mode: %s\n", programPath, mode)

	// Generate a session ID
	sessionID := fmt.Sprintf("session-%d", uuid.New().ID())

	// Start the Delve DAP server
	port := "54321" // Hardcoded for simplicity
	dlvCmd := exec.Command("dlv", "dap", "--listen=127.0.0.1:"+port)

	if err := dlvCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Delve DAP server: %w", err)
	}

	// Give the server a moment to start up
	time.Sleep(1 * time.Second)

	// Connect to the DAP server
	client := NewClient()
	err := client.Connect(context.Background(), "127.0.0.1:"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DAP server: %w", err)
	}

	// Initialize the debug session
	err = client.Initialize(programPath, args, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize debug session: %w", err)
	}

	// Create a new session
	session := &Session{
		id:       sessionID,
		client:   client,
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

	// Return session info
	return &common.SessionInfo{
		ID:          session.GetID(),
		ProgramPath: programPath,
		State:       "created",
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

// Session represents a DAP debug session
type Session struct {
	id       string
	client   common.DebuggerClient
	program  string
	cmd      *exec.Cmd
	isPaused bool
}

// GetID returns the session ID
func (s *Session) GetID() string {
	return s.id
}

// SetBreakpoint sets a breakpoint at the given file and line
func (s *Session) SetBreakpoint(file string, line int) (int, error) {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Setting breakpoint at %s:%d\n", file, line)

	// Construct breakpoint parameters
	params := map[string]interface{}{
		"file": file,
		"line": line,
	}

	// Send the request
	_, err := s.client.SendRequest("setBreakpoint", params)
	if err != nil {
		return 0, fmt.Errorf("failed to set breakpoint: %w", err)
	}

	// Extract breakpoint ID from response
	// This is just a placeholder - actual implementation would depend on the response format
	return 1, nil
}

// Continue continues execution until the next breakpoint
func (s *Session) Continue() error {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Continue called, current state: %s\n",
		map[bool]string{true: "paused", false: "running"}[s.isPaused])

	if !s.isPaused {
		return fmt.Errorf("cannot continue: program is not paused")
	}

	// Send continue request
	_, err := s.client.SendRequest("continue", nil)
	if err != nil {
		return fmt.Errorf("failed to continue execution: %w", err)
	}

	s.isPaused = false
	return nil
}

// Next steps over the current line
func (s *Session) Next() error {
	if !s.isPaused {
		return fmt.Errorf("cannot step: program is not paused")
	}

	// Send next request
	_, err := s.client.SendRequest("next", nil)
	if err != nil {
		return fmt.Errorf("failed to step over: %w", err)
	}

	return nil
}

// StepIn steps into the current function
func (s *Session) StepIn() error {
	if !s.isPaused {
		return fmt.Errorf("cannot step in: program is not paused")
	}

	// Send stepIn request
	_, err := s.client.SendRequest("stepIn", nil)
	if err != nil {
		return fmt.Errorf("failed to step in: %w", err)
	}

	return nil
}

// StepOut steps out of the current function
func (s *Session) StepOut() error {
	if !s.isPaused {
		return fmt.Errorf("cannot step out: program is not paused")
	}

	// Send stepOut request
	_, err := s.client.SendRequest("stepOut", nil)
	if err != nil {
		return fmt.Errorf("failed to step out: %w", err)
	}

	return nil
}

// Evaluate evaluates an expression in the current context
func (s *Session) Evaluate(expr string) (string, error) {
	fmt.Fprintf(os.Stderr, "DEBUG Session: Evaluating expression '%s' using debugger type: dap\n", expr)

	if !s.isPaused {
		return "", fmt.Errorf("cannot evaluate: program is not paused")
	}

	// Construct evaluate parameters
	params := map[string]interface{}{
		"expression": expr,
	}

	// Send the request
	_, err := s.client.SendRequest("evaluate", params)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate expression: %w", err)
	}

	// Extract result from response
	// This is just a placeholder - actual implementation would depend on the response format
	return "Expression result: (placeholder for DAP implementation)", nil
}

// Terminate terminates the debug session
func (s *Session) Terminate() error {
	// Send terminate request
	_, err := s.client.SendRequest("terminate", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to send terminate request: %v\n", err)
	}

	// Close the client connection
	if err := s.client.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to close client connection: %v\n", err)
	}

	// Kill the DAP server process
	if s.cmd != nil && s.cmd.Process != nil {
		if err := s.cmd.Process.Kill(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to kill DAP server process: %v\n", err)
		}
	}

	return nil
}

// IsPaused returns whether the debug session is paused
func (s *Session) IsPaused() bool {
	return s.isPaused
}

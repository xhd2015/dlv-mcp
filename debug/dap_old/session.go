package dap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/go-dap"
)

// SessionState represents the state of a debug session
type SessionState string

const (
	// SessionStateInitializing indicates the session is initializing
	SessionStateInitializing SessionState = "initializing"
	// SessionStateRunning indicates the session is running
	SessionStateRunning SessionState = "running"
	// SessionStatePaused indicates the session is paused
	SessionStatePaused SessionState = "paused"
	// SessionStateTerminated indicates the session is terminated
	SessionStateTerminated SessionState = "terminated"
)

// Session represents a debug session
type Session struct {
	ID            string
	ProgramPath   string
	Args          []string
	State         SessionState
	Client        *Client
	DlvCmd        *exec.Cmd
	BreakpointIDs map[string]int // Map of "file:line" to breakpoint ID
	mu            sync.Mutex
	created       time.Time
	debuggerType  string // Type of debugger: "dap" or "headless"
}

// SessionManager manages debug sessions
type SessionManager struct {
	sessions     map[string]*Session
	mu           sync.Mutex
	debuggerType string // Default debugger type: "dap" or "headless"
}

// NewSessionManager creates a new session manager
func NewSessionManager(debuggerType string) *SessionManager {
	// Default to headless if not specified
	if debuggerType == "" {
		debuggerType = "headless"
	}

	return &SessionManager{
		sessions:     make(map[string]*Session),
		debuggerType: debuggerType,
	}
}

// CreateSession creates a new debug session
func (m *SessionManager) CreateSession(ctx context.Context, programPath string, args []string, mode string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate session ID
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	// Create DAP client
	client := NewClient()

	// Create session
	session := &Session{
		ID:            sessionID,
		ProgramPath:   programPath,
		Args:          args,
		State:         SessionStateInitializing,
		Client:        client,
		BreakpointIDs: make(map[string]int),
		created:       time.Now(),
		debuggerType:  m.debuggerType,
	}

	// Start delve server based on debugger type
	port := "54321" // TODO: Find available port

	var dlvCmd *exec.Cmd
	if m.debuggerType == "dap" {
		// Use DAP mode
		fmt.Fprintf(os.Stderr, "DEBUG Session: Starting Delve in DAP mode\n")
		dlvCmd = exec.Command("dlv", "dap", "--listen=127.0.0.1:"+port)
	} else {
		// Use headless mode (better for variable evaluation)
		fmt.Fprintf(os.Stderr, "DEBUG Session: Starting Delve in headless mode\n")

		// Determine the appropriate command based on the mode
		dlvCommand := "debug"
		if mode == "exec" {
			dlvCommand = "exec"
		} else if mode == "test" {
			dlvCommand = "test"
		}

		// For headless mode, we need to specify the command (debug, exec, test)
		// and use the --headless flag
		dlvCmd = exec.Command("dlv", dlvCommand, "--headless", "--api-version=2", "--listen=127.0.0.1:"+port, programPath)

		// Add program arguments if provided
		if len(args) > 0 {
			dlvCmd.Args = append(dlvCmd.Args, "--")
			dlvCmd.Args = append(dlvCmd.Args, args...)
		}
	}

	session.DlvCmd = dlvCmd

	// Configure command
	session.DlvCmd.Stdout = os.Stderr // Redirect to stderr instead of stdout
	session.DlvCmd.Stderr = os.Stderr

	// Start the Delve server
	if err := session.DlvCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start Delve server: %w", err)
	}

	// Wait for server to start
	time.Sleep(500 * time.Millisecond)

	// Connect to server
	if err := client.Connect(ctx, "127.0.0.1:"+port); err != nil {
		session.DlvCmd.Process.Kill()
		return nil, err
	}

	// Initialize and launch based on debugger type
	var err error
	if m.debuggerType == "dap" {
		// DAP mode initialization
		if err = client.Initialize(); err != nil {
			session.DlvCmd.Process.Kill()
			client.Close()
			return nil, fmt.Errorf("failed to initialize debug session: %w", err)
		}

		// Launch program
		if err = client.Launch(programPath, args, mode); err != nil {
			session.DlvCmd.Process.Kill()
			client.Close()
			return nil, fmt.Errorf("failed to launch program: %w", err)
		}

		// Signal that configuration is done
		fmt.Fprintf(os.Stderr, "DEBUG Session: Calling ConfigurationDone\n")
		if err := client.ConfigurationDone(); err != nil {
			fmt.Fprintf(os.Stderr, "DEBUG Session: ConfigurationDone failed: %v\n", err)
			// Don't fail the session creation if this fails
		}
	} else {
		// Headless mode initialization
		if err = client.InitializeHeadless(programPath, args, mode); err != nil {
			session.DlvCmd.Process.Kill()
			client.Close()
			return nil, fmt.Errorf("failed to initialize headless debug session: %w", err)
		}
	}

	// Start a goroutine to listen for events
	go func() {
		for msg := range client.events {
			// Handle stopped events
			if stoppedEvent, ok := msg.(*dap.StoppedEvent); ok {
				fmt.Fprintf(os.Stderr, "DEBUG Session: Received StoppedEvent, reason: %s\n", stoppedEvent.Body.Reason)
				session.mu.Lock()
				session.State = SessionStatePaused
				fmt.Fprintf(os.Stderr, "DEBUG Session: State changed to %s\n", session.State)
				session.mu.Unlock()
			}

			// Handle terminated events
			if _, ok := msg.(*dap.TerminatedEvent); ok {
				fmt.Fprintf(os.Stderr, "DEBUG Session: Received TerminatedEvent\n")
				session.mu.Lock()
				session.State = SessionStateTerminated
				fmt.Fprintf(os.Stderr, "DEBUG Session: State changed to %s\n", session.State)
				session.mu.Unlock()
			}
		}
	}()

	// Update session state
	session.State = SessionStateRunning

	// Store session
	m.sessions[sessionID] = session

	return session, nil
}

// GetSession gets a session by ID
func (m *SessionManager) GetSession(id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found: %s", id)
	}

	return session, nil
}

// TerminateSession terminates a session
func (m *SessionManager) TerminateSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}

	// Close client connection
	if session.Client != nil {
		session.Client.Close()
	}

	// Kill delve process
	if session.DlvCmd != nil && session.DlvCmd.Process != nil {
		session.DlvCmd.Process.Kill()
	}

	// Update session state
	session.State = SessionStateTerminated

	// Remove session
	delete(m.sessions, id)

	return nil
}

// ListSessions lists all active sessions
func (m *SessionManager) ListSessions() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// SetBreakpoint sets a breakpoint in a session
func (s *Session) SetBreakpoint(file string, line int) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Fprintf(os.Stderr, "DEBUG Session: Setting breakpoint at %s:%d\n", file, line)

	// Check if breakpoint already exists
	key := fmt.Sprintf("%s:%d", file, line)
	if id, ok := s.BreakpointIDs[key]; ok {
		fmt.Fprintf(os.Stderr, "DEBUG Session: Breakpoint already exists with ID %d\n", id)
		return id, nil
	}

	// Set breakpoint
	id, err := s.Client.SetBreakpoint(file, line)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG Session: Failed to set breakpoint: %v\n", err)
		return 0, err
	}

	// Store breakpoint ID
	s.BreakpointIDs[key] = id
	fmt.Fprintf(os.Stderr, "DEBUG Session: Breakpoint set with ID %d\n", id)

	return id, nil
}

// Continue continues program execution
func (s *Session) Continue() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Fprintf(os.Stderr, "DEBUG Session: Continue called, current state: %s\n", s.State)

	// Remove this check since some debuggers might allow continue from non-paused state
	// if s.State != SessionStatePaused {
	// 	return fmt.Errorf("program is not paused")
	// }

	err := s.Client.Continue()
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG Session: Continue failed: %v\n", err)
		return err
	}

	s.State = SessionStateRunning
	fmt.Fprintf(os.Stderr, "DEBUG Session: State changed to %s\n", s.State)
	return nil
}

// Next steps over the current line
func (s *Session) Next() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionStatePaused {
		return fmt.Errorf("program is not paused")
	}

	err := s.Client.Next()
	if err != nil {
		return err
	}

	return nil
}

// StepIn steps into the current function call
func (s *Session) StepIn() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionStatePaused {
		return fmt.Errorf("program is not paused")
	}

	err := s.Client.StepIn()
	if err != nil {
		return err
	}

	return nil
}

// StepOut steps out of the current function
func (s *Session) StepOut() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionStatePaused {
		return fmt.Errorf("program is not paused")
	}

	err := s.Client.StepOut()
	if err != nil {
		return err
	}

	return nil
}

// Evaluate evaluates an expression
func (s *Session) Evaluate(expr string, frameId int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.State != SessionStatePaused {
		return "", fmt.Errorf("program is not paused")
	}

	// Log which debugger type we're using
	fmt.Fprintf(os.Stderr, "DEBUG Session: Evaluating expression '%s' using debugger type: %s\n", expr, s.debuggerType)

	// Evaluate the expression
	// Note: Headless mode should provide better variable evaluation capabilities
	// than DAP mode due to limitations in the DAP protocol implementation
	result, err := s.Client.Evaluate(expr, frameId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG Session: Evaluation failed: %v\n", err)
		return "", err
	}

	fmt.Fprintf(os.Stderr, "DEBUG Session: Evaluation result: %s\n", result)
	return result, nil
}

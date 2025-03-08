package common

import (
	"context"
)

// DebuggerClient is the interface that both DAP and headless clients must implement
type DebuggerClient interface {
	// Connect establishes a connection to the debug server
	Connect(ctx context.Context, addr string) error

	// Close closes the connection to the debug server
	Close() error

	// Initialize initializes the debug session
	Initialize(program string, args []string, mode string) error

	// SendRequest sends a request to the debug server and returns the response
	// Optional callback can be provided for asynchronous commands
	SendRequest(method string, params interface{}, callback ...chan interface{}) (interface{}, error)

	// IsClosed returns whether the client is closed
	IsClosed() bool
}

// SessionManager is the interface for managing debug sessions
type SessionManager interface {
	// NewSession creates a new debug session
	NewSession(programPath string, args []string, mode string) (Session, error)

	// GetDebuggerType returns the type of debugger being used
	GetDebuggerType() string

	// Methods needed by the tools package

	// CreateSession creates a new debug session with the given parameters
	CreateSession(ctx context.Context, programPath string, args []string, mode string) (*SessionInfo, error)

	// TerminateSession terminates a debug session
	TerminateSession(sessionID string) error

	// ListSessions returns a list of active debug sessions
	ListSessions() []*SessionInfo

	// GetSession returns a debug session by ID
	GetSession(sessionID string) (Session, error)
}

// Session is the interface for a debug session
type Session interface {
	// GetID returns the session ID
	GetID() string

	// SetBreakpoint sets a breakpoint at the given file and line
	SetBreakpoint(file string, line int) (int, error)

	// Continue continues execution until the next breakpoint
	Continue() error

	// Next steps over the current line
	Next() error

	// StepIn steps into the current function
	StepIn() error

	// StepOut steps out of the current function
	StepOut() error

	// Evaluate evaluates an expression in the current context
	Evaluate(expr string) (string, error)

	// Terminate terminates the debug session
	Terminate() error

	// IsPaused returns whether the debug session is paused
	IsPaused() bool
}

// SessionInfo holds information about a debug session
type SessionInfo struct {
	ID          string
	ProgramPath string
	State       string
}

package dap

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/google/go-dap"
	"github.com/xhd2015/debugger-mcp/debug/common"
)

// Client represents a DAP client that communicates with a Delve DAP server
type Client struct {
	conn     net.Conn
	reader   *bufio.Reader
	seq      int
	events   chan dap.Message
	isClosed bool
}

// NewClient creates a new DAP client
func NewClient() common.DebuggerClient {
	return &Client{
		seq:      1,
		events:   make(chan dap.Message, 100),
		isClosed: false,
	}
}

// Connect connects to a DAP server
func (c *Client) Connect(ctx context.Context, addr string) error {
	var d net.Dialer
	var err error

	// Set connection timeout to 10 seconds
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c.conn, err = d.DialContext(timeoutCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to DAP server: %w", err)
	}

	// Create buffered reader
	c.reader = bufio.NewReader(c.conn)

	// Start event listener
	go c.readEvents()

	return nil
}

// Close closes the connection to the DAP server
func (c *Client) Close() error {
	c.isClosed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsClosed returns whether the client is closed
func (c *Client) IsClosed() bool {
	return c.isClosed
}

// Initialize initializes a DAP debug session
func (c *Client) Initialize(program string, args []string, mode string) error {
	fmt.Fprintf(os.Stderr, "DEBUG: Initializing DAP debug session for program: %s, mode: %s\n", program, mode)

	// Initialize the debug adapter
	_, err := c.initialize()
	if err != nil {
		return fmt.Errorf("failed to initialize debug adapter: %w", err)
	}

	// Launch the program
	_, err = c.launch(program, args, mode)
	if err != nil {
		return fmt.Errorf("failed to launch program: %w", err)
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Successfully initialized and launched program\n")
	return nil
}

// SendRequest sends a request to the DAP server and returns the response
func (c *Client) SendRequest(method string, params interface{}, callback ...chan interface{}) (interface{}, error) {
	// DAP doesn't use the method name directly like headless; instead we map to proper DAP requests
	switch method {
	case "RPCServer.CreateBreakpoint":
		// Handle create breakpoint request
		// Extract file and line from params
		return nil, fmt.Errorf("not implemented in DAP mode")
	case "RPCServer.Command":
		// Handle command request
		if commandParams, ok := params.([]interface{}); ok && len(commandParams) > 0 {
			if commandMap, ok := commandParams[0].(map[string]interface{}); ok {
				if name, ok := commandMap["Name"].(string); ok && name == "continue" {
					// Handle continue command
					return nil, fmt.Errorf("not implemented in DAP mode")
				}
			}
		}
		return nil, fmt.Errorf("invalid command parameters")
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

// readEvents reads events from the DAP server
func (c *Client) readEvents() {
	for {
		if c.isClosed {
			break
		}

		message, err := c.readMessage()
		if err != nil {
			if err == io.EOF {
				// Connection closed
				c.isClosed = true
				close(c.events)
				break
			}
			fmt.Fprintf(os.Stderr, "ERROR reading from DAP server: %v\n", err)
			continue
		}

		// Process the message based on its type
		switch m := message.(type) {
		case *dap.OutputEvent:
			fmt.Fprintf(os.Stderr, "DAP Output: %s\n", m.Body.Output)
		case *dap.StoppedEvent:
			fmt.Fprintf(os.Stderr, "DEBUG: StoppedEvent received: reason=%s, threadId=%d\n", m.Body.Reason, m.Body.ThreadId)
		default:
			fmt.Fprintf(os.Stderr, "DAP message received: %T\n", message)
		}
	}
}

// readMessage reads a DAP message from the connection
func (c *Client) readMessage() (dap.Message, error) {
	// Implementation of DAP message reading
	// This would be based on the existing code but simplified for this example
	return nil, fmt.Errorf("not implemented")
}

// initialize sends an initialize request to the DAP server
func (c *Client) initialize() (*dap.InitializeResponse, error) {
	// Implementation of the initialize request
	return nil, fmt.Errorf("not implemented")
}

// launch sends a launch request to the DAP server
func (c *Client) launch(program string, args []string, mode string) (*dap.LaunchResponse, error) {
	// Implementation of the launch request
	return nil, fmt.Errorf("not implemented")
}

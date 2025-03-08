package headless

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// Simplified request structure for JSON-RPC
type jsonRPCRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	Id     int           `json:"id"`
}

// Simplified response structure for JSON-RPC
type jsonRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Id int `json:"id"`
}

// Client represents a headless client that communicates with a Delve headless server
type Client struct {
	conn           net.Conn
	reader         *bufio.Reader
	seq            int
	events         chan interface{}
	isClosed       bool
	addr           string        // Store the server address for reconnection
	mutex          sync.Mutex    // Protect concurrent access to connection
	reconnectDelay time.Duration // Delay between reconnection attempts
}

// NewClient creates a new headless client
func NewClient() *Client {
	return &Client{
		seq:            1,
		events:         make(chan interface{}, 100),
		isClosed:       false,
		reconnectDelay: 500 * time.Millisecond,
	}
}

// Connect connects to a headless server
func (c *Client) Connect(ctx context.Context, addr string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Store the address for reconnection
	c.addr = addr

	var d net.Dialer
	var err error

	// Set connection timeout to 10 seconds
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c.conn, err = d.DialContext(timeoutCtx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to headless server: %w", err)
	}

	// Create buffered reader
	c.reader = bufio.NewReader(c.conn)

	// Reset the closed flag
	c.isClosed = false

	// Start event listener in a new goroutine
	// go c.readEvents()

	fmt.Fprintf(os.Stderr, "DEBUG: Connected to Delve server at %s\n", addr)
	return nil
}

// Close closes the connection to the headless server
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

// Initialize initializes a headless debug session
func (c *Client) Initialize(program string, args []string, mode string) error {
	fmt.Fprintf(os.Stderr, "DEBUG: Initializing headless debug session for program: %s, mode: %s\n", program, mode)

	// Set up a proper command based on debug mode
	var debugCmd string
	if mode == "exec" {
		debugCmd = "exec"
	} else if mode == "test" {
		debugCmd = "test"
	} else {
		// Default to "debug"
		debugCmd = "debug"
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Headless debug mode: %s for program: %s\n", debugCmd, program)

	// Check that connection is established
	if c.reader == nil || c.conn == nil {
		return fmt.Errorf("connection to Delve headless server not established")
	}

	fmt.Fprintf(os.Stderr, "DEBUG: Successfully initialized headless debug session\n")
	return nil
}

// SendRequest sends a request to the headless server and returns the response
func (c *Client) SendRequest(method string, params interface{}, callback ...chan interface{}) (interface{}, error) {
	return sendRequest[any](c, RPCMethod(method), params, callback...)
}

func sendRequest[T any](c *Client, method RPCMethod, params interface{}, callback ...chan interface{}) (T, error) {
	var result T
	c.mutex.Lock()
	if c.isClosed {
		c.mutex.Unlock()
		return result, fmt.Errorf("client is closed")
	}

	if c.conn == nil {
		c.mutex.Unlock()
		return result, fmt.Errorf("connection to server not established")
	}

	// Increment sequence number under lock
	seqNum := c.seq
	c.seq++
	c.mutex.Unlock() // Release lock after reading shared state

	// Create a typed request structure
	req := jsonRPCRequest{
		Method: string(method),
		Id:     seqNum,
	}

	// Format params for Delve's JSON-RPC API
	// Detect if it's one of the known Delve RPC request types
	switch typedParams := params.(type) {
	case rpc2.CreateBreakpointIn:
		// Direct handling of CreateBreakpointIn
		req.Params = []interface{}{typedParams}
	case rpc2.EvalIn:
		// Direct handling of EvalIn
		req.Params = []interface{}{typedParams}
	case rpc2.StateIn:
		// Direct handling of StateIn
		req.Params = []interface{}{typedParams}
	case map[string]interface{}:
		// For standard map params, check for Breakpoint field
		if bp, exists := typedParams["Breakpoint"]; exists {
			if bpMap, ok := bp.(map[string]interface{}); ok {
				var breakpoint api.Breakpoint

				// Map breakpoint fields from untyped map to typed struct
				if file, ok := bpMap["file"].(string); ok {
					breakpoint.File = file
				}
				if line, ok := bpMap["line"].(float64); ok {
					breakpoint.Line = int(line)
				} else if line, ok := bpMap["line"].(int); ok {
					breakpoint.Line = line
				}

				// Replace untyped map with typed struct
				typedParams["Breakpoint"] = breakpoint
			}
		}
		req.Params = []interface{}{typedParams}
	default:
		// For any other parameter type
		req.Params = []interface{}{params}
	}

	// Serialize request to JSON
	requestBytes, err := json.Marshal(req)
	if err != nil {
		return result, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Log the request for debugging
	fmt.Fprintf(os.Stderr, "DEBUG: Sending request to Delve: %s\n", requestBytes)

	// Add newline for Delve headless server
	requestBytes = append(requestBytes, '\n')

	// Protect the send operation with a mutex
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check again if the client is closed
	if c.isClosed {
		return result, fmt.Errorf("client is closed")
	}

	// Send the request
	if _, err := c.conn.Write(requestBytes); err != nil {
		// On connection error, try to reconnect
		if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
			if reconnErr := c.reconnectLocked(context.Background()); reconnErr != nil {
				return result, fmt.Errorf("failed to send request and reconnect: %w", err)
			}
			// Retry after reconnection
			c.mutex.Unlock()
			return sendRequest[T](c, method, params, callback...)
		}
		return result, fmt.Errorf("failed to send request: %w", err)
	}

	// For asynchronous commands, return immediately
	if method == RPCCommand && len(callback) > 0 {
		fmt.Fprintf(os.Stderr, "DEBUG: Asynchronous command sent: %s\n", method)
		return result, nil
	}

	// Read the response directly
	// This is more reliable than using a goroutine for reading
	line, err := c.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
			// Connection error, try to reconnect
			if reconnErr := c.reconnectLocked(context.Background()); reconnErr != nil {
				return result, fmt.Errorf("connection failed and reconnect also failed: %w", reconnErr)
			}
			// Retry the request
			c.mutex.Unlock()
			return sendRequest[T](c, method, params, callback...)
		}
		return result, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the JSON response using our typed structure
	var resp jsonRPCResponse
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		return result, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for errors in the response
	if resp.Error != nil {
		return result, fmt.Errorf("error from Delve: %s", resp.Error.Message)
	}

	// Check if this is the response to our request
	if resp.Id != seqNum {
		return result, fmt.Errorf("response ID %d does not match request ID %d", resp.Id, seqNum)
	}

	// Unmarshal the result into the typed response
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return result, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return result, nil
}

// reconnectLocked attempts to reconnect to the Delve server
// Caller must hold the mutex lock
func (c *Client) reconnectLocked(ctx context.Context) error {
	// Close the existing connection if it's still open
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.reader = nil
	}

	if c.addr == "" {
		return fmt.Errorf("cannot reconnect: no server address stored")
	}

	// Attempt to reconnect
	fmt.Fprintf(os.Stderr, "DEBUG: Attempting to reconnect to Delve server at %s\n", c.addr)

	var d net.Dialer
	var err error

	// Set connection timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	c.conn, err = d.DialContext(timeoutCtx, "tcp", c.addr)
	if err != nil {
		return fmt.Errorf("failed to reconnect to headless server: %w", err)
	}

	// Create new buffered reader
	c.reader = bufio.NewReader(c.conn)

	// Reset the closed flag
	c.isClosed = false

	fmt.Fprintf(os.Stderr, "DEBUG: Successfully reconnected to Delve server\n")
	return nil
}

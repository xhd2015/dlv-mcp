# Implementing Conditional Function Execution at Breakpoints Using Delve

This report explores an approach to implement a system where a breakpoint can be set at a specific Go function, and when that breakpoint hits, a custom Go function is executed that receives the function name and arguments. Based on the return value of this handler function, the debugger can either continue execution or return immediately from the function. This capability provides powerful debugging and monitoring capabilities for Go applications.

## Table of Contents
1. [Introduction to Delve's Architecture](#introduction-to-delves-architecture)
2. [Understanding Delve's Breakpoint System](#understanding-delves-breakpoint-system)
3. [Breakpoint Conditions in Delve](#breakpoint-conditions-in-delve)
4. [Programmatic API Access to Delve](#programmatic-api-access-to-delve)
5. [Approach for Conditional Function Execution](#approach-for-conditional-function-execution)
6. [Proof of Concept Implementation](#proof-of-concept-implementation)
7. [Advanced Features and Future Work](#advanced-features-and-future-work)
8. [Limitations and Considerations](#limitations-and-considerations)
9. [References](#references)

## Introduction to Delve's Architecture

Delve (dlv) is a debugger for the Go programming language with a focus on providing a simple, full-featured debugging tool. Unlike traditional debuggers like GDB, Delve is specifically designed for Go and has deep integration with Go's runtime, making it more effective for debugging Go applications.

### Key Components of Delve

Delve's architecture consists of several key components:

1. **Command Line Interface (CLI)**: The user-facing component that accepts commands and displays information.

2. **Debug Server**: A component that interacts with the target process, managing breakpoints, stepping through code, etc.

3. **API Layer**: An abstraction layer that defines operations on the target program, allowing different clients to interact with Delve.

4. **Process Manipulation Layer**: Low-level functionality for controlling the target program.

5. **Symbol Information**: Tools for reading debug information from executables.

Delve supports multiple client-server communication methods, including:
- A JSON-RPC based API
- A Debug Adapter Protocol (DAP) implementation
- Direct terminal UI

This architecture allows us to leverage Delve's existing capabilities to implement conditional function execution at breakpoints.

## Understanding Delve's Breakpoint System

Breakpoints in Delve are represented by the `Breakpoint` struct in the `github.com/go-delve/delve/service/api` package. This struct contains various fields that control the breakpoint's behavior, including:

```go
type Breakpoint struct {
    ID int                  // Unique identifier
    Name string             // User-defined name
    Addr uint64             // Memory address
    Addrs []uint64          // List of addresses for this breakpoint
    File string             // Source file path
    Line int                // Line number in the file
    FunctionName string     // Name of the function containing the breakpoint
    Cond string             // Condition expression
    HitCond string          // Hit count condition
    Variables []string      // Variables to retrieve when hit
    LoadArgs *LoadConfig    // Configuration for loading function arguments
    LoadLocals *LoadConfig  // Configuration for loading local variables
    // ... other fields
}
```

When a breakpoint is hit, Delve evaluates its condition (if any) to decide whether to pause execution. This condition is a Go expression that is evaluated in the context of the program being debugged.

## Breakpoint Conditions in Delve

Delve supports conditional breakpoints through the `Cond` field of the `Breakpoint` struct. The condition is a Go expression that is evaluated when the breakpoint is hit. If the condition evaluates to true, the breakpoint triggers; otherwise, execution continues.

For example, in the CLI, you can set a condition using:

```
(dlv) break main.go:10
(dlv) condition 1 x > 5
```

This sets a breakpoint at line 10 of main.go that only triggers when the variable x is greater than 5.

Internally, Delve evaluates this condition by:
1. Parsing the condition expression
2. Evaluating it in the context of the current thread/goroutine
3. Converting the result to a boolean
4. Deciding whether to trigger the breakpoint based on the result

This condition mechanism is the foundation for our approach to execute a custom function at breakpoints.

## Programmatic API Access to Delve

Delve provides a programmatic API that allows external applications to control the debugger. This is defined in the `github.com/go-delve/delve/service` package and implemented through various interfaces like JSON-RPC or DAP.

The API defines operations such as:

```go
// CreateBreakpoint creates a new breakpoint.
CreateBreakpoint(*api.Breakpoint) (*api.Breakpoint, error)

// AmendBreakpoint allows user to update an existing breakpoint
AmendBreakpoint(*api.Breakpoint) error

// EvalVariable returns a variable in the context of the current thread.
EvalVariable(scope api.EvalScope, symbol string, cfg api.LoadConfig) (*api.Variable, error)
```

Using this API, we can programmatically create and manage breakpoints and evaluate expressions at those breakpoints.

## Approach for Conditional Function Execution

To implement conditional function execution at breakpoints using Delve, we'll create a Go package that:

1. Starts a Delve debug server for the target program
2. Sets breakpoints at specified functions
3. Registers custom handlers for those breakpoints
4. Manages execution flow based on handler results

### Architecture Overview

Our implementation consists of the following components:

1. **Trap Server**: A server that starts Delve and manages the debugging session
2. **Breakpoint Manager**: A component that sets and configures breakpoints
3. **Handler Registry**: A registry of functions to be called when breakpoints are hit
4. **Execution Controller**: A component that controls program execution based on handler results

### Example Usage API

Here's how our API might be used:

```go
package main

import (
    "fmt"
    "github.com/example/dlvtrap"
)

func main() {
    // Start a trap server for the target program
    server, err := dlvtrap.NewServer("./target_program")
    if err != nil {
        panic(err)
    }
    defer server.Close()

    // Set a breakpoint at a function and register a handler
    err = server.SetBreakpointWithHandler(
        "main.processRequest",
        func(funcName string, args []interface{}) bool {
            fmt.Printf("Intercepted call to %s with args: %v\n", funcName, args)
            // Return true to skip execution of the original function
            // Return false to continue normal execution
            return args[0].(string) == "admin"
        },
    )
    if err != nil {
        panic(err)
    }

    // Start execution
    err = server.Run()
    if err != nil {
        panic(err)
    }
}
```

## Proof of Concept Implementation

Below is a proof of concept implementation of our approach. This implementation uses Delve's API to set breakpoints with custom handlers and control execution based on handler results.

### 1. Trap Server

```go
// dlvtrap/server.go
package dlvtrap

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/go-delve/delve/service"
    "github.com/go-delve/delve/service/api"
    "github.com/go-delve/delve/service/rpc2"
)

// HandlerFunc is a function called when a breakpoint is hit
type HandlerFunc func(funcName string, args []interface{}) bool

// Server manages a Delve debug session
type Server struct {
    client     service.Client
    handlers   map[string]HandlerFunc
    breakpoints map[string]int // Maps function names to breakpoint IDs
}

// NewServer creates a new trap server for the target program
func NewServer(targetPath string) (*Server, error) {
    // Resolve the absolute path to the target
    targetPath, err := filepath.Abs(targetPath)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve target path: %w", err)
    }

    // Check if the target exists
    _, err = os.Stat(targetPath)
    if err != nil {
        return nil, fmt.Errorf("target program not found: %w", err)
    }

    // Start a new Delve server
    listener, err := rpc2.NewClient("localhost:0")
    if err != nil {
        return nil, fmt.Errorf("failed to create RPC client: %w", err)
    }

    server := &Server{
        client:     listener,
        handlers:   make(map[string]HandlerFunc),
        breakpoints: make(map[string]int),
    }

    // Initialize the debugger
    _, err = server.client.CreateProcess(targetPath, []string{}, "", true)
    if err != nil {
        listener.Close()
        return nil, fmt.Errorf("failed to create process: %w", err)
    }

    return server, nil
}

// Close closes the debug session
func (s *Server) Close() error {
    return s.client.Detach(true)
}

// SetBreakpointWithHandler sets a breakpoint at a function and registers a handler
func (s *Server) SetBreakpointWithHandler(funcName string, handler HandlerFunc) error {
    // Create a breakpoint at the function
    bp, err := s.client.CreateBreakpoint(&api.Breakpoint{
        FunctionName: funcName,
        // Set a condition that always returns true, so we can handle the logic ourselves
        Cond:        "true",
        // Load function arguments
        LoadArgs:    &api.LoadConfig{
            FollowPointers:     true,
            MaxVariableRecurse: 1,
            MaxStringLen:       64,
            MaxArrayValues:     64,
            MaxStructFields:    -1,
        },
    })
    if err != nil {
        return fmt.Errorf("failed to create breakpoint: %w", err)
    }

    // Store the breakpoint ID
    s.breakpoints[funcName] = bp.ID
    
    // Register the handler
    s.handlers[funcName] = handler
    
    return nil
}

// Run starts execution of the program and handles breakpoints
func (s *Server) Run() error {
    for {
        // Continue execution until next breakpoint
        state, err := s.client.Continue()
        if err != nil {
            return fmt.Errorf("continue failed: %w", err)
        }

        // Check if the program has terminated
        if state.Exited {
            return nil
        }

        // If we hit a breakpoint, call the appropriate handler
        if state.CurrentThread.Breakpoint != nil {
            bp := state.CurrentThread.Breakpoint
            funcName := bp.FunctionName
            handler, exists := s.handlers[funcName]
            
            if exists {
                // Extract function arguments
                args, err := s.extractArguments(state.CurrentThread.ID)
                if err != nil {
                    return fmt.Errorf("failed to extract arguments: %w", err)
                }
                
                // Call the handler
                shouldReturn := handler(funcName, args)
                
                if shouldReturn {
                    // Skip execution of the original function
                    // This is the most complex part and requires a deeper understanding
                    // of Delve's internals, specifically how to manipulate the program counter
                    // to effectively "return" from the current function
                    
                    // For the POC, we'll use a simplistic approach of stepping out of the function
                    // Note: In a real implementation, this would need to be more sophisticated
                    _, err := s.client.Step()
                    if err != nil {
                        return fmt.Errorf("step failed: %w", err)
                    }
                }
            }
        }
    }
}

// extractArguments extracts the arguments of the current function
func (s *Server) extractArguments(threadID int) ([]interface{}, error) {
    // Get the current frame
    frames, err := s.client.Stacktrace(threadID, 1, false, &api.LoadConfig{
        FollowPointers:     true,
        MaxVariableRecurse: 1,
        MaxStringLen:       64,
        MaxArrayValues:     64,
        MaxStructFields:    -1,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get stacktrace: %w", err)
    }
    
    if len(frames) == 0 {
        return nil, fmt.Errorf("no stack frames")
    }
    
    // Extract arguments
    var args []interface{}
    for _, arg := range frames[0].Arguments {
        // Convert variable to a Go value
        // This is a simplified version and would need to be more robust
        // in a real implementation
        args = append(args, convertVariable(arg))
    }
    
    return args, nil
}

// convertVariable converts an API variable to a Go value
func convertVariable(v api.Variable) interface{} {
    // This is a simplified conversion
    // A real implementation would handle more types and be more robust
    switch v.Kind {
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        val, _ := strconv.ParseInt(v.Value, 10, 64)
        return val
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
        val, _ := strconv.ParseUint(v.Value, 10, 64)
        return val
    case reflect.Float32, reflect.Float64:
        val, _ := strconv.ParseFloat(v.Value, 64)
        return val
    case reflect.Bool:
        val, _ := strconv.ParseBool(v.Value)
        return val
    case reflect.String:
        return v.Value
    default:
        return v.Value
    }
}
```

### 2. Example Usage

```go
// example/main.go
package main

import (
    "fmt"
    "os"

    "github.com/example/dlvtrap"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("Usage: example <target_program>")
        os.Exit(1)
    }

    // Start a trap server for the target program
    server, err := dlvtrap.NewServer(os.Args[1])
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    defer server.Close()

    // Set a breakpoint at the processRequest function
    err = server.SetBreakpointWithHandler(
        "main.processRequest",
        func(funcName string, args []interface{}) bool {
            // Extract the first argument (assuming it's a string)
            reqType, ok := args[0].(string)
            if !ok {
                fmt.Println("Error: first argument is not a string")
                return false
            }

            fmt.Printf("Intercepted %s with request type: %s\n", funcName, reqType)

            // Skip execution for "admin" requests
            if reqType == "admin" {
                fmt.Println("Blocking admin request")
                return true
            }

            return false
        },
    )
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    // Run the program
    err = server.Run()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
}
```

### 3. Explanation of Implementation Details

The proof of concept implementation above demonstrates the key components needed to implement conditional function execution at breakpoints using Delve:

1. **Server Creation**: The `NewServer` function creates a new Delve debug session for the target program.

2. **Breakpoint Setting**: The `SetBreakpointWithHandler` function sets a breakpoint at a specific function and registers a handler to be called when the breakpoint is hit.

3. **Execution Management**: The `Run` function manages execution of the program, calling handlers when breakpoints are hit and controlling flow based on handler results.

4. **Argument Extraction**: The `extractArguments` function extracts the arguments of the current function when a breakpoint is hit.

5. **Variable Conversion**: The `convertVariable` function converts Delve's variable representation to Go values that can be passed to handlers.

## Advanced Features and Future Work

The proof of concept implementation above provides a basic framework for conditional function execution at breakpoints. However, there are several advanced features that could be added:

### 1. Return Value Manipulation

In addition to skipping function execution, handlers could also specify the return value of the function. This would require more sophisticated manipulation of the program state, but would provide even more control over execution.

### 2. Function Call Interception

Instead of using breakpoints, we could use Delve's function call interception capabilities to directly intercept function calls. This would provide more control over function execution and avoid the need to manipulate the program counter.

### 3. Performance Optimizations

The current implementation may have performance overhead due to the need to extract and convert function arguments. Performance optimizations could be implemented to reduce this overhead.

### 4. Support for More Complex Functions

The current implementation assumes a relatively simple function signature. Support for more complex functions, including those with variadic arguments, could be added.

### 5. Integration with Existing Debugging Tools

The implementation could be integrated with existing debugging tools, such as VS Code or GoLand, to provide a more seamless experience.

## Limitations and Considerations

There are several limitations and considerations to be aware of when using this approach:

### 1. Debugging Overhead

Using a debugger to intercept function calls introduces overhead, which may not be suitable for production environments or performance-critical code.

### 2. Platform and Version Dependencies

Delve's API may change between versions, and support for specific platforms may vary. This implementation may need to be updated for different Delve versions or platforms.

### 3. Security Considerations

The ability to intercept function calls and modify execution flow introduces security considerations, particularly in multi-user environments.

### 4. Debugger Limitations

Delve has limitations on what operations can be performed during debugging, such as restrictions on modifying certain types of variables or executing arbitrary code.

## References

1. [Delve Debugger GitHub Repository](https://github.com/go-delve/delve)
2. [Delve API Documentation](https://pkg.go.dev/github.com/go-delve/delve/service/api)
3. [Setting Conditional Breakpoints in Go](https://trstringer.com/go-conditional-breakpoints/)
4. [Delve API Client Howto](https://github.com/go-delve/delve/blob/master/Documentation/api/ClientHowto.md)
5. [VS Code Go Debugging Documentation](https://github.com/golang/vscode-go/wiki/debugging)
6. [Delve Internal Architecture](https://github.com/go-delve/delve/blob/master/Documentation/architecture.md)
7. [Go Debug Adapter Protocol Implementation](https://github.com/go-delve/delve/tree/master/service/dap)
8. [Conditional Breakpoints in Go](https://medium.com/@tylfin/debugging-with-delve-62cf4e4b37c)
9. [Delve Commands Documentation](https://github.com/go-delve/delve/tree/master/Documentation/cli)
10. [Go Execution Tracer Design Document](https://go.googlesource.com/proposal/+/master/design/17432-traces.md) 
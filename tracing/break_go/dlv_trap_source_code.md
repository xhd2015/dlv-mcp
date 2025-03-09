# Implementing Conditional Function Execution by Modifying Delve's Source Code

This report explores the possibility of directly modifying Delve's source code to implement a system where a handler function is executed when a breakpoint is hit. This handler function would receive information about the function being executed and its arguments, and based on the handler's return value, the debugger could either continue execution or skip the original function.

## Table of Contents
1. [Introduction](#introduction)
2. [Understanding Delve's Architecture](#understanding-delves-architecture)
3. [Delve's Breakpoint System](#delves-breakpoint-system)
4. [Proposed Modifications](#proposed-modifications)
5. [Proof of Concept Implementation](#proof-of-concept-implementation)
6. [Integration Points](#integration-points)
7. [Limitations and Considerations](#limitations-and-considerations)
8. [Advanced Features](#advanced-features)
9. [Testing and Validation](#testing-and-validation)
10. [References](#references)

## Introduction

The goal of this modification is to extend Delve's functionality to allow breakpoints that execute a custom Go function when hit. This would enable powerful debugging and monitoring capabilities, such as:

- Function call interception and modification
- Runtime monitoring of specific functions
- Conditional execution based on arguments
- Function execution tracing with custom logic
- Security and validation of function arguments before execution

While Delve already supports conditional breakpoints through boolean expression evaluation, extending it to call arbitrary Go functions requires deeper modifications to its source code.

## Understanding Delve's Architecture

Delve is organized into several key components:

1. **Command Line Interface (CLI)**: The user interface that processes commands.
2. **Service Layer**: Provides a JSON-RPC or DAP (Debug Adapter Protocol) API for clients.
3. **Debugger Package**: Implements debugging operations like setting breakpoints, stepping, etc.
4. **Process Control (pkg/proc)**: Low-level functionality for controlling target programs.
5. **Binary Analysis**: Tools for reading debug information from executables.

The main components relevant to our modification are:

- `pkg/proc/breakpoints.go`: Defines breakpoint types and condition evaluation logic
- `service/debugger`: Implements high-level debugging operations
- `service/api`: Defines the API exposed to clients
- `service/rpc2`: Implements the JSON-RPC server

## Delve's Breakpoint System

### Breakpoint Types

Delve uses a layered approach to breakpoints:

1. **Physical Breakpoints**: Represent actual breakpoints at memory addresses.
2. **Breaklets**: Multiple logical breakpoints can share a single physical breakpoint.
3. **Logical Breakpoints**: User-defined breakpoints with conditions, variables to load, etc.

### Key Structures

The key structures in Delve's breakpoint system are:

```go
// Breakpoint represents a physical breakpoint
type Breakpoint struct {
    FunctionName string
    File         string
    Line         int
    Addr         uint64
    OriginalData []byte
    
    // ... other fields
    
    Breaklets []*Breaklet
    Logical   *LogicalBreakpoint
}

// Breaklet represents one of multiple breakpoints that can overlap
type Breaklet struct {
    Kind      BreakpointKind
    LogicalID int
    Cond      ast.Expr
    // ... other fields
}

// LogicalBreakpoint contains user-specified information about a breakpoint
type LogicalBreakpoint struct {
    ID            int
    Name          string
    Addr          uint64
    Addrs         []uint64
    HitCount      map[string]uint64
    TotalHitCount uint64
    Cond          string
    HitCond       string
    // ... other fields
}
```

### Condition Evaluation

Breakpoint conditions are evaluated using the `evalBreakpointCondition` function:

```go
func evalBreakpointCondition(tgt *Target, thread Thread, cond ast.Expr) (bool, error) {
    if cond == nil {
        return true, nil
    }
    scope, err := GoroutineScope(tgt, thread)
    if err != nil {
        scope, err = ThreadScope(tgt, thread)
        if err != nil {
            return true, err
        }
    }
    v, err := scope.evalAST(cond)
    if err != nil {
        return true, fmt.Errorf("error evaluating expression: %v", err)
    }
    if v.Kind != reflect.Bool {
        return true, errors.New("condition expression not boolean")
    }
    v.loadValue(loadFullValue)
    if v.Unreadable != nil {
        return true, fmt.Errorf("condition expression unreadable: %v", v.Unreadable)
    }
    return constant.BoolVal(v.Value), nil
}
```

When a breakpoint is hit, Delve checks if the condition evaluates to true before stopping execution.

## Proposed Modifications

### Overview

To implement handler function execution on breakpoints, we need to:

1. Extend the `LogicalBreakpoint` and `Breakpoint` structures to include a handler function reference
2. Modify the breakpoint condition evaluation process to call the handler
3. Add a mechanism to register and manage handler functions
4. Extend the API to allow clients to set breakpoints with handlers
5. Implement function argument extraction and passing to the handler

### Extended Structures

We would need to extend the existing structures:

```go
// Extended LogicalBreakpoint
type LogicalBreakpoint struct {
    // ... existing fields
    
    HandlerFunc     string // Name of the registered handler function
    SkipExecution   bool   // Whether to skip execution of the original function
}

// Handler function type
type BreakpointHandler func(funcName string, args []interface{}) bool
```

### Handler Registry

We would need to add a new component to register and manage handler functions:

```go
// HandlerRegistry manages registered handler functions
type HandlerRegistry struct {
    mutex    sync.RWMutex
    handlers map[string]BreakpointHandler
}

// RegisterHandler registers a new handler function
func (r *HandlerRegistry) RegisterHandler(name string, handler BreakpointHandler) {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    r.handlers[name] = handler
}

// GetHandler returns a registered handler function
func (r *HandlerRegistry) GetHandler(name string) (BreakpointHandler, bool) {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    handler, ok := r.handlers[name]
    return handler, ok
}
```

## Proof of Concept Implementation

Below is a proof of concept implementation showing the key modifications needed to enable handler function execution on breakpoints:

### 1. Extending the Breakpoint Structure

```go
// pkg/proc/breakpoints.go

// Breakpoint represents a physical breakpoint
type Breakpoint struct {
    // ... existing fields
    
    // HandlerFunc is the name of the registered handler function
    HandlerFunc string
    
    // SkipExecution indicates whether to skip the execution of the original function
    SkipExecution bool
}
```

### 2. Creating a Handler Registry

```go
// pkg/proc/handler.go

package proc

import (
    "fmt"
    "sync"
)

// BreakpointHandler is a function called when a breakpoint is hit
type BreakpointHandler func(funcName string, args []interface{}) bool

// HandlerRegistry manages registered handler functions
type HandlerRegistry struct {
    mutex    sync.RWMutex
    handlers map[string]BreakpointHandler
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
    return &HandlerRegistry{
        handlers: make(map[string]BreakpointHandler),
    }
}

// RegisterHandler registers a new handler function
func (r *HandlerRegistry) RegisterHandler(name string, handler BreakpointHandler) {
    r.mutex.Lock()
    defer r.mutex.Unlock()
    r.handlers[name] = handler
}

// GetHandler returns a registered handler function
func (r *HandlerRegistry) GetHandler(name string) (BreakpointHandler, bool) {
    r.mutex.RLock()
    defer r.mutex.RUnlock()
    handler, ok := r.handlers[name]
    return handler, ok
}
```

### 3. Modifying the Breakpoint Handling

```go
// pkg/proc/breakpoints.go

// Extend the checkCondition function to call the handler
func (bp *Breakpoint) checkCondition(tgt *Target, thread Thread, bpstate *BreakpointState) {
    // ... existing code
    
    // If there's a handler function, extract arguments and call it
    if bp.HandlerFunc != "" {
        handler, ok := tgt.HandlerRegistry.GetHandler(bp.HandlerFunc)
        if ok {
            // Extract function arguments
            args, err := extractFunctionArguments(tgt, thread)
            if err != nil {
                // Log error and continue with normal execution
                fmt.Printf("Error extracting function arguments: %v\n", err)
            } else {
                // Call the handler
                skipExecution := handler(bp.FunctionName, args)
                if skipExecution {
                    bp.SkipExecution = true
                }
            }
        }
    }
    
    // ... continue with normal condition checking
}

// Function to extract arguments of the current function
func extractFunctionArguments(tgt *Target, thread Thread) ([]interface{}, error) {
    scope, err := ThreadScope(tgt, thread)
    if err != nil {
        return nil, err
    }
    
    // Get the current function and its arguments
    frames, err := ThreadStacktrace(tgt, thread, 1)
    if err != nil || len(frames) < 1 {
        return nil, fmt.Errorf("could not get stacktrace: %v", err)
    }
    
    frame := frames[0]
    var args []interface{}
    
    // Extract all function arguments
    for _, arg := range frame.Arguments {
        // Convert the variable to a Go value
        value, err := convertVariableToInterface(arg)
        if err != nil {
            continue
        }
        args = append(args, value)
    }
    
    return args, nil
}

// Convert a Variable to a Go interface{}
func convertVariableToInterface(v *Variable) (interface{}, error) {
    v.loadValue(loadFullValue)
    if v.Unreadable != nil {
        return nil, v.Unreadable
    }
    
    switch v.Kind {
    case reflect.Bool:
        return constant.BoolVal(v.Value), nil
    case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
        return constant.Int64Val(v.Value)
    case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
        return constant.Uint64Val(v.Value)
    case reflect.Float32, reflect.Float64:
        return constant.Float64Val(v.Value)
    case reflect.String:
        return constant.StringVal(v.Value), nil
    // ... handle other types
    default:
        return v.Value.String(), nil
    }
}
```

### 4. Modifying the Continue Operation

```go
// pkg/proc/target_exec.go

// Modify the Continue function to check if execution should be skipped
func (t *Target) Continue() error {
    // ... existing code
    
    // After a breakpoint is hit
    if bp := currentThread.Breakpoint(); bp != nil && bp.SkipExecution {
        // Skip execution by stepping over the function
        return t.StepOut()
    }
    
    // ... continue with normal execution
}
```

### 5. Extending the API

```go
// service/api/types.go

// Extend the Breakpoint struct in the API
type Breakpoint struct {
    // ... existing fields
    
    // HandlerFunc is the name of the registered handler function
    HandlerFunc string `json:"handlerFunc,omitempty"`
}
```

```go
// service/rpc2/server.go

// Add a new RPC method to register handler functions
type RegisterHandlerIn struct {
    Name string
    // The handler function code as a string, to be compiled and executed
    Handler string
}

type RegisterHandlerOut struct {
}

func (s *RPCServer) RegisterHandler(arg RegisterHandlerIn, out *RegisterHandlerOut) error {
    // Compile the handler function code
    handlerFunc, err := compileHandlerFunction(arg.Handler)
    if err != nil {
        return err
    }
    
    // Register the handler
    s.debugger.RegisterHandler(arg.Name, handlerFunc)
    return nil
}

// Function to compile a handler function from a string
func compileHandlerFunction(code string) (proc.BreakpointHandler, error) {
    // This would involve using go/parser and go/types to parse and type-check the code,
    // then using reflect or a package like yaegi to create a callable function
    // For a POC, we could use a simpler approach with pre-registered handlers
    
    // Placeholder implementation
    return func(funcName string, args []interface{}) bool {
        // Default implementation
        return false
    }, nil
}
```

## Integration Points

The key integration points for this modification would be:

1. **Target Structure**: Add a `HandlerRegistry` field to the `Target` struct.
2. **API Extension**: Add methods to register and manage handler functions.
3. **Breakpoint Evaluation**: Modify the breakpoint evaluation logic to call handlers.
4. **Function Skipping**: Implement a mechanism to skip function execution based on handler results.

### Target Structure Modification

```go
// pkg/proc/target.go

type Target struct {
    // ... existing fields
    
    // HandlerRegistry manages breakpoint handlers
    HandlerRegistry *HandlerRegistry
}

// Initialize the HandlerRegistry in the appropriate constructors
func NewTarget() *Target {
    t := &Target{
        // ... existing initialization
        
        HandlerRegistry: NewHandlerRegistry(),
    }
    return t
}
```

## Limitations and Considerations

### Technical Challenges

1. **Function Skipping**: Efficiently skipping function execution without disturbing the stack could be challenging.
2. **Argument Extraction**: Correctly extracting and converting function arguments for all types would require thorough implementation.
3. **Handler Management**: Securely compiling and executing user-provided handler functions would require careful implementation.
4. **Performance Impact**: The overhead of calling handlers on frequently hit breakpoints could impact performance.

### Security Considerations

1. **Code Execution**: Allowing arbitrary code execution in handlers presents security risks.
2. **Process State Modification**: Handlers could potentially corrupt the process state if not carefully implemented.
3. **Privilege Escalation**: Handlers might access sensitive information in the target process.

### Compatibility

1. **Go Version Compatibility**: The implementation should work with different Go versions.
2. **Platform Compatibility**: The implementation should work across supported platforms (Linux, Windows, macOS).
3. **API Compatibility**: Changes should maintain backward compatibility with existing clients.

## Advanced Features

### Return Value Modification

A more advanced extension would be to allow handlers to specify the return value when skipping function execution:

```go
// Extended handler function type
type BreakpointHandlerWithReturn func(funcName string, args []interface{}) (skip bool, returnValues []interface{})

// This would require modifying the stack to "fake" a function return with the specified values
```

### Function Call Interception

Instead of skipping functions, allow replacing their implementation entirely:

```go
// Extended handler function type
type BreakpointHandlerWithReplacement func(funcName string, args []interface{}) (replace bool, returnValues []interface{})

// This would require a mechanism to execute the handler in the context of the target process
```

### Tracing and Logging

Add built-in handlers for common tasks like tracing and logging:

```go
// TraceHandler traces function calls and arguments
func TraceHandler(funcName string, args []interface{}) bool {
    fmt.Printf("TRACE: %s(%v)\n", funcName, args)
    return false // continue execution
}

// Register the trace handler by default
registry.RegisterHandler("trace", TraceHandler)
```

## Testing and Validation

To validate the implementation, we would need:

1. **Unit Tests**: Tests for the handler registry and argument extraction functions.
2. **Integration Tests**: Tests that set breakpoints with handlers and verify correct behavior.
3. **Benchmark Tests**: Tests to measure the performance impact of handlers.
4. **Cross-Platform Tests**: Tests to verify the implementation works across supported platforms.

## References

1. [Delve GitHub Repository](https://github.com/go-delve/delve)
2. [Delve Architecture Documentation](https://github.com/go-delve/delve/blob/master/Documentation/architecture.md)
3. [Delve API Documentation](https://pkg.go.dev/github.com/go-delve/delve/service/api)
4. [Go Debug Adapter Protocol Implementation](https://github.com/go-delve/delve/tree/master/service/dap)
5. [Breakpoints in Delve](https://github.com/go-delve/delve/blob/master/pkg/proc/breakpoints.go)
6. [Conditional Breakpoints in Go](https://trstringer.com/go-conditional-breakpoints/)
7. [Delve Commands Documentation](https://github.com/go-delve/delve/tree/master/Documentation/cli)
8. [Go Execution Tracer Design Document](https://go.googlesource.com/proposal/+/master/design/17432-traces.md)
9. [Delve Issue: Conditional Breakpoints](https://github.com/go-delve/delve/issues/75)
10. [Delve Issue: Step-over improvement](https://github.com/go-delve/delve/pull/1876) 
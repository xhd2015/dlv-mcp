# Delve MCP Server Demo

This demo shows how to use the Delve MCP Server to debug a Go program using the Model Context Protocol (MCP).

## What this Demo Does

The demo program (`main.go`) demonstrates a complete debugging workflow:

1. Starts the MCP server as a child process
2. Starts a debug session for the `./testdata/hello.go` program
3. Sets a breakpoint at the `add` function (line 21)
4. Continues execution to hit the breakpoint
5. Evaluates variables at the breakpoint
6. Steps through code
7. Continues to program completion
8. Terminates the debug session

## How to Run the Demo

Make sure you're in the root directory of the project, then run:

```bash
go run ./demo/main.go
```

## Expected Output

The demo will output logs showing:

- The MCP server starting
- Commands being sent to the server
- Responses from the server
- Debug session actions (setting breakpoints, evaluating variables, etc.)

## Understanding the Code

The demo showcases:

1. **Starting the MCP Server**: Using `exec.Command` to start the server as a child process
2. **Communicating with MCP**: Sending JSON-formatted requests to the server's stdin and reading responses from stdout
3. **Debug Workflow**: Demonstrating a complete debugging session

## Extending the Demo

You can modify the demo to:

- Debug different Go programs
- Set multiple breakpoints
- Add more complex evaluation expressions
- Implement a more interactive debugging experience 
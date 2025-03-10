# Go Delve Debugger MCP Server

An MCP server that allows LLMs to interact with the Delve Go debugger.

## Description

This project implements a [Model Context Protocol (MCP)](https://github.com/modelcontextprotocol/servers) server that integrates with the [Delve Go debugger](https://github.com/go-delve/delve) to enable AI assistants to debug Go applications. The server exposes a set of tools that allow LLMs to start, control, and interact with debug sessions.

You can find a working example at [EXAMPLE.md](./EXAMPLE.md).

## Features

- Start debug sessions for Go programs
- Set breakpoints in code
- Step through code (next, step in, step out)
- Evaluate expressions in the context of a debug session
- Inspect variables and stack traces
- Manage multiple debug sessions

## Installation

### Prerequisites

- Go 1.21 or higher
- Delve debugger installed (`go install github.com/go-delve/delve/cmd/dlv@latest`)

### Install

```sh
go install github.com/xhd2015/dlv-mcp/cmd/dlv-mcp@latest
```

## Usage

Start the MCP server:

```sh
dlv-mcp --listen :9097
```

Then configure MCP Server at `http://localhost:9097/sse`, in Cursor or any MCP client.

### Inspect the MCP Server
```sh
bunx @modelcontextprotocol/inspector dlv-mcp
```

## Available Tools

### Debug Session Management

- `start_debug`: Start a new debug session
  - `program`: Path to Go program to debug
  - `args`: Command line arguments for the program (optional)
  - `mode`: Debug mode (`debug`, `test`, or `exec`, default: `debug`)

- `terminate_debug`: Terminate a debug session
  - `session_id`: ID of the debug session to terminate

- `list_debug_sessions`: List active debug sessions

### Breakpoint Management

- `set_breakpoint`: Set a breakpoint in a debug session
  - `session_id`: ID of the debug session
  - `file`: Source file to set breakpoint in (absolute path)
  - `line`: Line number to set breakpoint at

### Execution Control

- `continue`: Continue execution in a debug session
  - `session_id`: ID of the debug session

- `next`: Step over current line in a debug session
  - `session_id`: ID of the debug session

- `step_in`: Step into function in a debug session
  - `session_id`: ID of the debug session

- `step_out`: Step out of function in a debug session
  - `session_id`: ID of the debug session

### Inspection

- `evaluate`: Evaluate an expression in a debug session
  - `session_id`: ID of the debug session
  - `expression`: Expression to evaluate
  - `frame_id`: Stack frame ID (optional, default: 0)

## Example Workflow

1. Start a debug session:
   ```
   start_debug program=/path/to/program.go
   ```

2. Set a breakpoint:
   ```
   set_breakpoint session_id=session-123456 file=/path/to/program.go line=15
   ```

3. Continue to breakpoint:
   ```
   continue session_id=session-123456
   ```

4. Evaluate an expression:
   ```
   evaluate session_id=session-123456 expression="myVariable"
   ```

5. Step over a line:
   ```
   next session_id=session-123456
   ```

6. Terminate the session:
   ```
   terminate_debug session_id=session-123456
   ```

## Architecture

The server is built using the following components:

- **MCP Server Layer**: Implements the Model Context Protocol using mark3labs/mcp-go
- **DAP Client Layer**: Communicates with Delve's DAP server
- **Session Management**: Maintains and manages debug sessions

## Inspect The MCP Server
```sh
bunx @modelcontextprotocol/inspector go run ./cmd/dlv-mcp
```

## Using the Demo Client

We've created a demo client that shows how to use the debugger-mcp server programmatically. The demo:

1. Starts the MCP server
2. Launches a debug session for `./testdata/hello.go`
3. Tries to set a breakpoint and execute debugging commands
4. Properly terminates the session and server

To run the demo:

```sh
go run ./demo/main.go
```

The demo client showcases how to:
- Start and communicate with the MCP server using JSON-RPC
- Send tool calls using the correct format
- Parse and handle responses
- Deal with errors gracefully

Check out `demo/main.go` and `demo/README.md` for more details on the implementation.

## License

MIT License 
# Extended Debug Commands

This package adds the following additional debugger commands to the MCP server:

## Stack and Frame Operations

- **stacktrace**: Get call stack for the current goroutine or a specific goroutine
  - Parameters: `session_id`, `goroutine_id` (optional), `depth` (optional)

- **switch_goroutine**: Switch to a different goroutine
  - Parameters: `session_id`, `goroutine_id`

- **switch_thread**: Switch to a different thread
  - Parameters: `session_id`, `thread_id`

## Breakpoint Management

- **list_breakpoints**: List all breakpoints in the current debug session
  - Parameters: `session_id`

- **toggle_breakpoint**: Enable or disable a breakpoint
  - Parameters: `session_id`, `breakpoint_id`

- **clear_breakpoint**: Remove a breakpoint
  - Parameters: `session_id`, `breakpoint_id`

- **create_watchpoint**: Create a watchpoint on a variable
  - Parameters: `session_id`, `variable`, `scope` (optional), `write` (optional), `read` (optional)

## Variable Inspection

- **list_local_vars**: List local variables in the current stack frame
  - Parameters: `session_id`, `frame_id` (optional)

- **list_function_args**: List function arguments in the current stack frame
  - Parameters: `session_id`, `frame_id` (optional)

- **set_variable**: Change the value of a variable
  - Parameters: `session_id`, `scope`, `name`, `value`

- **examine_memory**: Examine memory at a specific address
  - Parameters: `session_id`, `address`, `length` (optional)

## Execution Control

- **restart**: Restart the debugged program
  - Parameters: `session_id`, `rebuild` (optional)

- **detach**: Detach from the debugged program and allow it to continue running
  - Parameters: `session_id`, `kill` (optional)

- **disassemble**: Show disassembly of the program
  - Parameters: `session_id`, `function` (optional), `file` (optional), `line` (optional), `pc` (optional)

## Checkpoint Management

- **create_checkpoint**: Create a checkpoint at the current position in the program
  - Parameters: `session_id`, `name` (optional)

- **list_checkpoints**: List all checkpoints in the current debug session
  - Parameters: `session_id`

- **clear_checkpoint**: Remove a checkpoint
  - Parameters: `session_id`, `checkpoint_id`

These commands extend the core debugging functionality to provide a more comprehensive debugging experience. 
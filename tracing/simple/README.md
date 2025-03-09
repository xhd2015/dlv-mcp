# Function Call Tracing Demo

This is a simple demonstration of function call tracing in Go programs using Delve as the underlying debugger. The demo consists of a sample program with nested function calls and a tracing solution that uses Delve to capture function entry and exit points, including arguments and return values.

## Demo Structure

- `target/main.go` - A sample Go program with 3 levels of nested function calls
- `trace_script.dlv` - The Delve script used for tracing function calls
- `trace_demo.go` - A Go program that:
  1. Reads the trace script and copies it to the target directory
  2. Builds the target program
  3. Executes the tracing with Delve and displays the results

This separation of concerns makes it easier to understand and modify each component independently.

## What the Demo Shows

When you run the demo, you'll see:

1. **Function Entry Tracing**: The demo captures when functions are called and what arguments are passed.
2. **Function Exit Tracing**: The demo captures when functions return and what values they return.
3. **Call Hierarchy**: You can observe the nested function call pattern through the trace output.

This demonstrates the essential capabilities needed for function call tracing in Go programs.

## How to Run the Demo

Simply run:

```
go run tracing/simple/trace_demo.go
```

## Sample Output

The output will show the program execution with tracing information:

```
Tracing function calls (showing entry and exit points with arguments/return values):
---------------------------------------------------------------------
> goroutine(1): main.processPerson(main.Person {Name: "Alice", Age: 30}, 2)
Processing person: Alice
> goroutine(1): main.multiplyAge(30, 2)
Multiplying age 30 by factor 2
> goroutine(1): main.calculateValue(30, 2)
Calculating value for 30 and 2
>> goroutine(1): main.calculateValue => ((unreadable empty OP stack),(unreadable empty OP stack))
>> goroutine(1): main.multiplyAge => ((unreadable empty OP stack),(unreadable empty OP stack))
>> goroutine(1): main.processPerson => ((unreadable empty OP stack),(unreadable empty OP stack))
Result: Value=60, Ok=true
...
```

In this output:
- Lines with `>` show function entries with arguments
- Lines with `>>` show function returns with return values
- The regular output shows the program's standard output

## How It Works: Approach 2 Implementation

This demo implements "Approach 2: Custom Script/Tool Extending Delve" from the PRD. It works by:

1. **Programmatic Setup**: Uses Go to programmatically set up the demo environment
2. **Dynamic Tracing**: Creates a Delve script that sets tracepoints on target functions
3. **Non-intrusive Tracing**: Captures function calls without modifying the target code

## Limitations and Challenges

The current implementation has some limitations:

1. **Return Value Capture**: We set breakpoints at return points, but Delve doesn't always properly display return values (shown as "unreadable empty OP stack").
2. **Error Handling**: The demo shows errors in script execution. In a production tool, these would need proper handling.
3. **Template Rendering**: Delve has limitations in its template syntax for the `on` command.

## Next Steps

To build a more robust function tracing tool as outlined in the PRD, the next steps would be:

1. **Direct API Integration**: Use Delve's API directly instead of scripts
2. **Structured Output**: Create structured representations of function calls
3. **Call Tree Visualization**: Implement proper call tree building with timing
4. **Performance Optimization**: Handle large volumes of function calls efficiently

## References

- [Delve Documentation](https://github.com/go-delve/delve/tree/master/Documentation)
- [Delve Command API](https://github.com/go-delve/delve/blob/master/Documentation/cli/README.md)
- [Delve API Documentation](https://github.com/go-delve/delve/tree/master/Documentation/api) 
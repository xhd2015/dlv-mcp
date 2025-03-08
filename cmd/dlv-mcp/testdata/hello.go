package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("Hello, Debugger!")

	// Print arguments if provided
	if len(os.Args) > 1 {
		fmt.Println("Arguments:", os.Args[1:])
	}

	// Simple function to set a breakpoint on
	result := add(5, 7)
	time.Sleep(10 * time.Hour)
	fmt.Printf("5 + 7 = %d\n", result)
}

func add(a, b int) int {
	// Good place to set a breakpoint
	return a + b
}

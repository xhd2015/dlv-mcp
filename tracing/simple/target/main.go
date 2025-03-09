package main

import (
	"fmt"
	"time"
)

// Global depth counter for tracking function calls

// Simple structs for demonstration
type Person struct {
	Name string
	Age  int
}

type Result struct {
	Value int
	Ok    bool
}

// Level 1 function
func processPerson(person Person, factor int) (Result, error) {
	time.Sleep(100 * time.Millisecond) // Simulate some work

	// Call level 2 function
	multiplied, err := multiplyAge(person.Age, factor)
	if err != nil {
		// Print exit message with depth and decrement depth counter
		return Result{}, fmt.Errorf("error in processPerson: %w", err)
	}

	result := Result{Value: multiplied, Ok: true}

	return result, nil
}

// Level 2 function
func multiplyAge(age, factor int) (int, error) {
	time.Sleep(50 * time.Millisecond) // Simulate some work

	// Call level 3 function
	result, err := calculateValue(age, factor)
	if err != nil {
		return 0, fmt.Errorf("error in multiplyAge: %w", err)
	}

	return result, nil
}

// Level 3 function
func calculateValue(a, b int) (int, error) {
	time.Sleep(25 * time.Millisecond) // Simulate some work

	if b == 0 {
		return 0, fmt.Errorf("cannot use factor 0")
	}

	result := a * b
	return result, nil
}

func main() {
	fmt.Println("Starting the program...")

	// Create a person
	person := Person{
		Name: "Alice",
		Age:  30,
	}

	// Process with different factors
	factors := []int{2, 3, 0, 5}

	for _, factor := range factors {
		fmt.Printf("\nProcessing with factor %d:\n", factor)
		result, err := processPerson(person, factor)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Result: Value=%d, Ok=%v\n", result.Value, result.Ok)
		}
	}

	fmt.Println("\nProgram completed")
}

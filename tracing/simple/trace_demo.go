package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Get the absolute path to the target directory
	workspaceDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Set up paths
	simpleDir := filepath.Join(workspaceDir, "tracing", "simple")
	targetDir := filepath.Join(simpleDir, "target")

	// Read the Delve script from trace_script.dlv
	scriptSrcPath := filepath.Join(simpleDir, "trace_script.dlv")
	scriptContent, err := os.ReadFile(scriptSrcPath)
	if err != nil {
		log.Fatalf("Failed to read script file %s: %v", scriptSrcPath, err)
	}

	// Create the Delve script file in the target directory
	scriptFile := filepath.Join(targetDir, "trace.dlv")
	err = os.WriteFile(scriptFile, scriptContent, 0644)
	if err != nil {
		log.Fatalf("Failed to write Delve script: %v", err)
	}

	// Build the program
	fmt.Println("Building the target program...")
	buildCmd := exec.Command("go", "build", "-o", "target", "main.go")
	buildCmd.Dir = targetDir // Set working directory to targetDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		log.Fatalf("Failed to build target program: %v", err)
	}

	fmt.Println("\nBuild completed successfully!")
	fmt.Println("\nRunning the tracing demo with Delve:")
	fmt.Printf("cd %s && dlv exec ./target --init trace.dlv\n", filepath.Base(targetDir))

	// Execute the Delve command
	fmt.Println("\n---------------------------------------------------------------------")

	// We'll use a simple timeout approach
	dlvCmd := exec.Command("dlv", "exec", "./target", "--init", "trace.dlv", "--allow-non-terminal-interactive=true", "-r", "stdin:/dev/null")
	dlvCmd.Dir = targetDir
	dlvCmd.Stdout = os.Stdout
	dlvCmd.Stderr = os.Stderr

	// Start Delve
	err = dlvCmd.Start()
	if err != nil {
		log.Fatalf("Failed to start Delve: %v", err)
	}

	err = dlvCmd.Wait()
	if err != nil {
		log.Fatalf("Delve exited with error: %v", err)
	}

	fmt.Println("\nDemo completed! The trace output above shows how function calls were traced.")
}

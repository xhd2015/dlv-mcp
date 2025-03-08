package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// findProjectRoot attempts to find the root directory of the project
func findProjectRoot(t *testing.T) string {
	// Start from the current working directory
	dir, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")

	// Walk up directories looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			t.Fatal("Could not find project root with go.mod")
			return ""
		}
		dir = parent
	}
}

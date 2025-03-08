package headless_ext

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-delve/delve/service/rpc2"
	"github.com/xhd2015/dlv-mcp/debug/common"
	"github.com/xhd2015/dlv-mcp/debug/headless"
)

// ListSources lists all source files in the debugged program matching the filter
// Uses the RPCServer.ListSources API method:
// https://pkg.go.dev/github.com/go-delve/delve/service/rpc2#RPCServer.ListSources
func ListSources(session common.Session, filter string) (string, error) {
	// Create the request using the official API type
	listSourcesIn := rpc2.ListSourcesIn{
		Filter: filter,
	}

	// Call the ListSources RPC method
	listSourcesOut, err := sendHeadlessClientRequest[rpc2.ListSourcesOut](
		session,
		headless.RPCListSources,
		listSourcesIn,
	)
	if err != nil {
		return "", fmt.Errorf("failed to list source files: %w", err)
	}

	// Format the response
	var builder strings.Builder
	if filter != "" {
		builder.WriteString(fmt.Sprintf("Source files matching filter '%s':\n", filter))
	} else {
		builder.WriteString("All source files:\n")
	}

	if len(listSourcesOut.Sources) == 0 {
		builder.WriteString("No source files found.")
		return builder.String(), nil
	}

	// Sort the sources for consistent output
	sources := listSourcesOut.Sources
	sort.Strings(sources)

	// Group source files by directory for better readability
	dirMap := make(map[string][]string)
	for _, source := range sources {
		dir := filepath.Dir(source)
		filename := filepath.Base(source)
		dirMap[dir] = append(dirMap[dir], filename)
	}

	// Sort directories for consistent output
	dirs := make([]string, 0, len(dirMap))
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	// Output files grouped by directory
	for _, dir := range dirs {
		files := dirMap[dir]
		builder.WriteString(fmt.Sprintf("\n%s/\n", dir))
		for _, file := range files {
			builder.WriteString(fmt.Sprintf("  %s\n", file))
		}
	}

	// Add a summary
	builder.WriteString(fmt.Sprintf("\nTotal: %d source files\n", len(sources)))
	return builder.String(), nil
}

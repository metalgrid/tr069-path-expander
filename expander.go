// Package expander provides TR-069 wildcard expansion functionality.
// It manages the iterative discovery process for parameter paths containing wildcards,
// maintaining internal state across multiple discovery rounds.
package expander

import (
	"errors"
	"strconv"
	"strings"
)

// Expander manages the expansion of TR-069 parameter paths containing wildcards.
// It maintains internal state across discovery iterations and provides methods
// to retrieve discovery paths and register discovered indices.
type Expander interface {
	// NextDiscoveryPath returns the next path segment for discovery.
	// Returns (path, true) if there's a path to discover, ("", false) if complete.
	NextDiscoveryPath() (string, bool)

	// RegisterParameterNames registers discovered parameter names for a given path.
	// Path should match the path returned by NextDiscoveryPath (without trailing dot).
	// parameterNames should be the actual parameter names returned by TR-069 GetParameterNames.
	RegisterParameterNames(path string, parameterNames []string) error

	// IsComplete returns true when all wildcards have been expanded.
	IsComplete() bool

	// ExpandedPaths returns all fully expanded parameter paths.
	// Only valid when IsComplete() returns true.
	ExpandedPaths() []string
}

// wildcardExpander implements the Expander interface.
type wildcardExpander struct {
	originalPath         string
	pathSegments         []string
	wildcardLevels       []int
	currentLevel         int
	discoveredPaths      map[string][]int
	pendingPaths         []string
	completedPaths       []string
	isComplete           bool
	expectedFinalPaths   map[string]bool // Tracks final-level paths that need registration
	registeredFinalPaths map[string]bool // Tracks final-level paths that have been registered
}

// pathState represents the state of a path during expansion.
type pathState struct {
	segments []string
	indices  []int
	level    int
}

// Common errors returned by the expander.
var (
	ErrInvalidPath     = errors.New("invalid path format")
	ErrPathMismatch    = errors.New("path mismatch")
	ErrAlreadyComplete = errors.New("expansion is already complete")
)

// parseWildcardPath parses a wildcard path and returns segments and wildcard positions.
func parseWildcardPath(path string) ([]string, []int, error) {
	if path == "" {
		return nil, nil, ErrInvalidPath
	}

	segments := strings.Split(path, ".")
	var wildcardLevels []int

	for i, segment := range segments {
		if segment == "*" {
			wildcardLevels = append(wildcardLevels, i)
		}
	}

	if len(wildcardLevels) == 0 {
		return segments, wildcardLevels, nil
	}

	return segments, wildcardLevels, nil
}

// buildDiscoveryPath constructs a discovery path up to the specified wildcard level.
func buildDiscoveryPath(segments []string, wildcardLevel int) string {
	var builder strings.Builder

	for i := 0; i < wildcardLevel; i++ {
		if i > 0 {
			builder.WriteByte('.')
		}
		builder.WriteString(segments[i])
	}

	builder.WriteByte('.')
	return builder.String()
}

// buildFullPath constructs a complete path by replacing wildcards with indices.
func buildFullPath(segments []string, wildcardLevels []int, indices []int) string {
	result := make([]string, len(segments))
	copy(result, segments)

	for i, level := range wildcardLevels {
		if i < len(indices) {
			result[level] = string(rune('0' + indices[i]))
		}
	}

	return strings.Join(result, ".")
}

// extractIndicesFromParameterNames extracts indices from TR-069 parameter names.
// For example, given path "Device.WiFi.AccessPoint" and parameter names:
// ["Device.WiFi.AccessPoint.1", "Device.WiFi.AccessPoint.2", "Device.WiFi.AccessPoint.3"]
// it returns [1, 2, 3].
func extractIndicesFromParameterNames(basePath string, parameterNames []string) []int {
	var indices []int
	basePathWithDot := basePath + "."

	for _, paramName := range parameterNames {
		// Check if parameter name starts with the base path
		if !strings.HasPrefix(paramName, basePathWithDot) {
			continue
		}

		// Extract the part after the base path
		suffix := paramName[len(basePathWithDot):]

		// Find the first segment (the index)
		parts := strings.Split(suffix, ".")
		if len(parts) == 0 {
			continue
		}

		// Try to parse the first part as an integer
		if index, err := strconv.Atoi(parts[0]); err == nil {
			// Check if this index is already in our list
			found := false
			for _, existing := range indices {
				if existing == index {
					found = true
					break
				}
			}
			if !found {
				indices = append(indices, index)
			}
		}
	}

	return indices
}

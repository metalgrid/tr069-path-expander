package expander

import (
	"fmt"
	"strconv"
	"strings"
)

// newWildcardExpander creates a new wildcard expander for the given path.
func newWildcardExpander(wildcardPath string) (*wildcardExpander, error) {
	segments, wildcardLevels, err := parseWildcardPath(wildcardPath)
	if err != nil {
		return nil, err
	}

	exp := &wildcardExpander{
		originalPath:         wildcardPath,
		pathSegments:         segments,
		wildcardLevels:       wildcardLevels,
		currentLevel:         0,
		discoveredPaths:      make(map[string][]int),
		pendingPaths:         make([]string, 0),
		completedPaths:       make([]string, 0),
		isComplete:           len(wildcardLevels) == 0,
		expectedFinalPaths:   make(map[string]bool),
		registeredFinalPaths: make(map[string]bool),
	}

	// If no wildcards, the path is already complete
	if len(wildcardLevels) == 0 {
		exp.completedPaths = append(exp.completedPaths, wildcardPath)
	} else {
		// Initialize with the first discovery path
		firstPath := buildDiscoveryPath(segments, wildcardLevels[0])
		exp.pendingPaths = append(exp.pendingPaths, firstPath)
	}

	return exp, nil
}

// NextDiscoveryPath returns the next path segment for discovery.
func (e *wildcardExpander) NextDiscoveryPath() (string, bool) {
	e.checkCompletion()

	if e.isComplete || len(e.pendingPaths) == 0 {
		return "", false
	}

	// Return the first pending path
	path := e.pendingPaths[0]
	e.pendingPaths = e.pendingPaths[1:]

	return path, true
}

// RegisterParameterNames registers discovered parameter names for a given path.
// It extracts indices from the parameter names automatically.
func (e *wildcardExpander) RegisterParameterNames(path string, parameterNames []string) error {
	// Extract indices from parameter names
	indices := extractIndicesFromParameterNames(path, parameterNames)

	// Allow registration of expected final paths even if expansion appears complete
	// This handles the case where pendingPaths is empty but we still need final registrations
	if e.isComplete && !e.expectedFinalPaths[path] {
		return ErrAlreadyComplete
	}

	// Validate that this path is expected
	if !e.isValidRegistrationPath(path) {
		return fmt.Errorf("%w: got %s", ErrPathMismatch, path)
	}

	// Store the discovered indices
	e.discoveredPaths[path] = indices

	// Generate next level paths or complete the expansion
	if err := e.processRegistration(path, indices); err != nil {
		return err
	}

	return nil
}

// IsComplete returns true when all wildcards have been expanded.
func (e *wildcardExpander) IsComplete() bool {
	e.checkCompletion()
	return e.isComplete
}

// ExpandedPaths returns all fully expanded parameter paths.
func (e *wildcardExpander) ExpandedPaths() []string {
	if !e.isComplete {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]string, len(e.completedPaths))
	copy(result, e.completedPaths)
	return result
}

// reset clears the expander state for reuse.
func (e *wildcardExpander) reset(wildcardPath string) error {
	segments, wildcardLevels, err := parseWildcardPath(wildcardPath)
	if err != nil {
		return err
	}

	e.originalPath = wildcardPath
	e.pathSegments = segments
	e.wildcardLevels = wildcardLevels
	e.currentLevel = 0

	// Clear maps and slices
	for k := range e.discoveredPaths {
		delete(e.discoveredPaths, k)
	}
	for k := range e.expectedFinalPaths {
		delete(e.expectedFinalPaths, k)
	}
	for k := range e.registeredFinalPaths {
		delete(e.registeredFinalPaths, k)
	}
	e.pendingPaths = e.pendingPaths[:0]
	e.completedPaths = e.completedPaths[:0]
	e.isComplete = len(wildcardLevels) == 0

	// Initialize state
	if len(wildcardLevels) == 0 {
		e.completedPaths = append(e.completedPaths, wildcardPath)
	} else {
		firstPath := buildDiscoveryPath(segments, wildcardLevels[0])
		e.pendingPaths = append(e.pendingPaths, firstPath)
	}

	return nil
}

// isValidRegistrationPath checks if a path is expected for registration.
func (e *wildcardExpander) isValidRegistrationPath(path string) bool {
	pathSegments := strings.Split(path, ".")

	// Find which wildcard level this path corresponds to
	for _, wildcardLevel := range e.wildcardLevels {
		if len(pathSegments) == wildcardLevel {
			// Check if the path matches the pattern up to this wildcard
			matches := true
			for i := 0; i < wildcardLevel; i++ {
				if i >= len(pathSegments) || i >= len(e.pathSegments) {
					matches = false
					break
				}
				// For wildcard positions, we accept any value (including numbers)
				// For non-wildcard positions, they must match exactly
				if e.pathSegments[i] != "*" && pathSegments[i] != e.pathSegments[i] {
					matches = false
					break
				}
			}
			if matches {
				return true
			}
		}
	}

	return false
}

// processRegistration handles the registration of indices and generates next steps.
func (e *wildcardExpander) processRegistration(path string, indices []int) error {
	pathSegments := strings.Split(path, ".")
	currentWildcardLevel := len(pathSegments)

	// Find the index of this wildcard level in our wildcardLevels slice
	wildcardIndex := -1
	for i, level := range e.wildcardLevels {
		if level == currentWildcardLevel {
			wildcardIndex = i
			break
		}
	}

	if wildcardIndex == -1 {
		return fmt.Errorf("internal error: wildcard level not found")
	}

	// If this is the last wildcard level, generate final paths
	if wildcardIndex == len(e.wildcardLevels)-1 {
		e.generateFinalPaths(path, indices)
		// Mark this path as registered in final paths tracking (only for multi-level)
		if len(e.expectedFinalPaths) > 0 {
			e.registeredFinalPaths[path] = true
		}
		e.checkCompletion()
		return nil
	}

	// Generate paths for the next wildcard level
	nextWildcardLevel := e.wildcardLevels[wildcardIndex+1]

	// If the next level is the final wildcard level, populate expectedFinalPaths
	if wildcardIndex+1 == len(e.wildcardLevels)-1 {
		for _, index := range indices {
			nextPath := e.buildNextLevelPath(path, index, nextWildcardLevel)
			// Remove trailing dot for the expected path key
			expectedPath := strings.TrimSuffix(nextPath, ".")
			e.expectedFinalPaths[expectedPath] = true
			e.pendingPaths = append(e.pendingPaths, nextPath)
		}
	} else {
		// Not the final level, just add to pending paths
		for _, index := range indices {
			nextPath := e.buildNextLevelPath(path, index, nextWildcardLevel)
			e.pendingPaths = append(e.pendingPaths, nextPath)
		}
	}
	return nil
}

// generateFinalPaths creates the final expanded paths for a given base path and indices.
func (e *wildcardExpander) generateFinalPaths(basePath string, indices []int) {
	// For each index, create a complete path by replacing all wildcards
	for _, index := range indices {
		finalPath := e.buildCompletePath(basePath, index)
		e.completedPaths = append(e.completedPaths, finalPath)
	}
}

// buildCompletePath constructs a complete path by replacing the current wildcard with an index.
func (e *wildcardExpander) buildCompletePath(basePath string, index int) string {
	baseSegments := strings.Split(basePath, ".")
	result := make([]string, len(e.pathSegments))
	copy(result, e.pathSegments)

	// Replace wildcards with discovered indices
	currentWildcardLevel := len(baseSegments)

	// Replace all wildcards up to the current level with discovered indices
	for _, wildcardLevel := range e.wildcardLevels {
		if wildcardLevel < currentWildcardLevel {
			// Use the index from the base path
			if wildcardLevel < len(baseSegments) {
				result[wildcardLevel] = baseSegments[wildcardLevel]
			}
		} else if wildcardLevel == currentWildcardLevel {
			// Use the current index
			result[wildcardLevel] = strconv.Itoa(index)
			break
		}
	}

	return strings.Join(result, ".")
}

// buildNextLevelPath constructs a path for the next wildcard level.
func (e *wildcardExpander) buildNextLevelPath(basePath string, index int, nextWildcardLevel int) string {
	// Start with the base path (without trailing dot) and add the current index
	baseWithoutDot := strings.TrimSuffix(basePath, ".")
	var builder strings.Builder
	builder.WriteString(baseWithoutDot)
	builder.WriteByte('.')
	builder.WriteString(strconv.Itoa(index))

	// Add segments up to the next wildcard (but not including the wildcard itself)
	currentLevel := len(strings.Split(baseWithoutDot, "."))
	for i := currentLevel; i < nextWildcardLevel; i++ {
		if i < len(e.pathSegments) && e.pathSegments[i] != "*" {
			builder.WriteByte('.')
			builder.WriteString(e.pathSegments[i])
		}
	}

	builder.WriteByte('.')
	return builder.String()
}

// checkCompletion determines if the expansion process is complete.
func (e *wildcardExpander) checkCompletion() {
	noPendingPaths := len(e.pendingPaths) == 0

	// If no expected final paths are set, use old logic (single wildcard case)
	if len(e.expectedFinalPaths) == 0 {
		e.isComplete = noPendingPaths
		return
	}

	// Multi-level case: check both conditions
	allFinalPathsRegistered := len(e.expectedFinalPaths) == len(e.registeredFinalPaths)
	e.isComplete = noPendingPaths && allFinalPathsRegistered
}

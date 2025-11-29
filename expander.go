// Package expander provides optimized TR-069 wildcard path expansion with automatic
// ancestor detection and caching to minimize redundant discoveries.
package expander

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Expander manages the expansion of TR-069 parameter paths containing wildcards.
// It automatically detects common ancestors to minimize discovery requests and
// maintains an internal cache of discovered indices for reuse.
type Expander struct {
	// paths stores all paths that need expansion, organized by their common ancestors
	paths pathTree

	// cache stores discovered indices for each discovery path to avoid redundant requests
	cache map[string][]int

	// pendingDiscoveries is a queue of discovery paths that need to be processed
	pendingDiscoveries []string

	// processedDiscoveries tracks which discovery paths have been processed
	processedDiscoveries map[string]bool

	// expandedPaths stores the final fully expanded parameter paths
	expandedPaths []string

	// expandedSet prevents duplicates in expandedPaths
	expandedSet map[string]bool

	// isComplete indicates if all discoveries have been processed
	isComplete bool

	// lastDiscoveryPath tracks the last discovery path returned by Next()
	lastDiscoveryPath string
}

// pathNode represents a node in the path tree structure
type pathNode struct {
	segment    string
	children   map[string]*pathNode
	isWildcard bool
	isLeaf     bool
	leafNames  []string // Store original leaf names for final expansion
}

// pathTree represents the tree structure of all paths to be expanded
type pathTree struct {
	root *pathNode
}

// Common errors returned by the expander
var (
	ErrInvalidPath     = errors.New("invalid path format")
	ErrEmptyResults    = errors.New("results cannot be empty")
	ErrNoDiscovery     = errors.New("no discovery path available")
	ErrAlreadyComplete = errors.New("expansion is already complete")
)

// Add adds one or more paths for expansion. Paths can be added at any time,
// and the expander will reuse its cache for common ancestors.
// Duplicate paths are automatically handled and won't appear twice in the output.
func (e *Expander) Add(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	// Mark as not complete since we're adding new paths
	e.isComplete = false

	for _, path := range paths {
		if path == "" {
			return ErrInvalidPath
		}

		// Add path to the tree structure
		if err := e.paths.addPath(path); err != nil {
			return fmt.Errorf("failed to add path %s: %w", path, err)
		}
	}

	// Generate discovery paths for newly added paths
	e.generateDiscoveryPaths()

	return nil
}

// Next returns the next discovery path that needs to be queried via GetParameterNames.
// Returns (path, true) if there's a path to discover, ("", false) if complete.
// The returned path includes a trailing dot for partial path discovery.
func (e *Expander) Next() (string, bool) {
	// Check if we have any pending discoveries
	for len(e.pendingDiscoveries) > 0 {
		path := e.pendingDiscoveries[0]
		e.pendingDiscoveries = e.pendingDiscoveries[1:]

		// Skip if already processed (might happen with dynamic additions)
		if e.processedDiscoveries[path] {
			continue
		}

		// Check if we have this in cache
		if _, cached := e.cache[path]; cached {
			// Mark as processed and continue to next
			e.processedDiscoveries[path] = true
			e.processNextLevel(path, e.cache[path])
			continue
		}

		// Store last discovery path and return it
		e.lastDiscoveryPath = path
		return path, true
	}

	// No more discoveries needed
	e.isComplete = true
	e.generateExpandedPaths()
	return "", false
}

// Register registers the discovered parameter names from a GetParameterNames call.
// The results should be the raw parameter names returned by the TR-069 device.
func (e *Expander) Register(results []string) error {
	if e.isComplete {
		return ErrAlreadyComplete
	}

	// Use the last discovery path from Next()
	discoveryPath := e.lastDiscoveryPath
	if discoveryPath == "" {
		return fmt.Errorf("no discovery path available - call Next() first")
	}

	// Extract indices from the results
	indices := extractIndices(discoveryPath, results)

	// Cache the results
	e.cache[discoveryPath] = indices
	e.processedDiscoveries[discoveryPath] = true

	// Process next level of discoveries based on these indices
	e.processNextLevel(discoveryPath, indices)

	// Clear last discovery path
	e.lastDiscoveryPath = ""

	return nil
}

// Collect returns all fully expanded parameter paths.
// This should be called after Next() returns false.
func (e *Expander) Collect() ([]string, error) {
	// Trigger final generation if not yet complete
	if !e.isComplete {
		// Check if there are truly pending discoveries
		path, hasMore := e.Next()
		if hasMore {
			return nil, fmt.Errorf("expansion not complete, next discovery path: %s", path)
		}
	}

	// Return a copy to prevent external modification
	result := make([]string, len(e.expandedPaths))
	copy(result, e.expandedPaths)
	return result, nil
}

// Reset clears all state in the expander, preparing it for reuse.
// This is automatically called when an expander is returned to the pool.
func (e *Expander) Reset() {
	// Clear the path tree
	e.paths.root = &pathNode{
		children: make(map[string]*pathNode),
	}

	// Clear all maps
	for k := range e.cache {
		delete(e.cache, k)
	}
	for k := range e.processedDiscoveries {
		delete(e.processedDiscoveries, k)
	}
	for k := range e.expandedSet {
		delete(e.expandedSet, k)
	}

	// Clear slices
	e.pendingDiscoveries = e.pendingDiscoveries[:0]
	e.expandedPaths = e.expandedPaths[:0]

	e.isComplete = false
	e.lastDiscoveryPath = ""
}

// generateDiscoveryPaths analyzes the path tree and generates discovery paths
// for all wildcard positions that haven't been processed yet
func (e *Expander) generateDiscoveryPaths() {
	discoveries := e.paths.getDiscoveryPaths()

	for _, disc := range discoveries {
		// Only add if not already processed or pending
		if !e.processedDiscoveries[disc] {
			// Check if already in pending
			found := false
			for _, pending := range e.pendingDiscoveries {
				if pending == disc {
					found = true
					break
				}
			}
			if !found {
				e.pendingDiscoveries = append(e.pendingDiscoveries, disc)
			}
		}
	}
}

// processNextLevel generates new discovery paths based on discovered indices
func (e *Expander) processNextLevel(discoveryPath string, indices []int) {
	// Build paths for the next wildcard level based on these indices
	nextPaths := e.paths.getNextLevelPaths(discoveryPath, indices)

	for _, nextPath := range nextPaths {
		// Only add if not already processed
		if !e.processedDiscoveries[nextPath] {
			// Check if already in pending
			found := false
			for _, pending := range e.pendingDiscoveries {
				if pending == nextPath {
					found = true
					break
				}
			}
			if !found {
				e.pendingDiscoveries = append(e.pendingDiscoveries, nextPath)
			}
		}
	}
}

// generateExpandedPaths creates the final fully expanded paths from the tree and cache
func (e *Expander) generateExpandedPaths() {
	// Don't clear existing paths - we might be adding dynamically
	// Generate all possible expanded paths from the tree using the cache
	paths := e.paths.generateExpandedPaths(e.cache)

	// Add unique paths only
	for _, path := range paths {
		if !e.expandedSet[path] {
			e.expandedPaths = append(e.expandedPaths, path)
			e.expandedSet[path] = true
		}
	}

	// Sort for consistent output
	sort.Strings(e.expandedPaths)
}

// extractIndices extracts numeric indices from parameter names
func extractIndices(discoveryPath string, parameterNames []string) []int {
	indices := []int{}
	seen := make(map[int]bool)

	pathWithoutDot := strings.TrimSuffix(discoveryPath, ".")
	prefixLen := len(pathWithoutDot) + 1 // +1 for the dot

	for _, param := range parameterNames {
		if !strings.HasPrefix(param, pathWithoutDot+".") {
			continue
		}

		// Extract the part after the prefix
		remainder := param[prefixLen:]

		// Find the next segment (up to the next dot or end)
		nextDot := strings.Index(remainder, ".")
		segment := remainder
		if nextDot != -1 {
			segment = remainder[:nextDot]
		}

		// Try to parse as integer
		if idx, err := strconv.Atoi(segment); err == nil {
			if !seen[idx] {
				indices = append(indices, idx)
				seen[idx] = true
			}
		}
	}

	// Sort indices for consistent ordering
	sort.Ints(indices)
	return indices
}

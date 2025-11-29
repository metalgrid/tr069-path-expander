package expander

import (
	"strconv"
	"strings"
)

// addPath adds a path to the tree structure
func (t *pathTree) addPath(path string) error {
	if t.root == nil {
		t.root = &pathNode{
			children: make(map[string]*pathNode),
		}
	}

	segments := strings.Split(path, ".")
	current := t.root

	for i, segment := range segments {
		if current.children == nil {
			current.children = make(map[string]*pathNode)
		}

		child, exists := current.children[segment]
		if !exists {
			child = &pathNode{
				segment:    segment,
				children:   make(map[string]*pathNode),
				isWildcard: segment == "*",
				isLeaf:     i == len(segments)-1,
			}
			current.children[segment] = child
		}

		// Mark as leaf if this is the last segment
		if i == len(segments)-1 {
			child.isLeaf = true
		}

		current = child
	}

	return nil
}

// getDiscoveryPaths returns all discovery paths needed for wildcards in the tree
func (t *pathTree) getDiscoveryPaths() []string {
	if t.root == nil {
		return nil
	}

	var paths []string
	t.collectDiscoveryPaths(t.root, "", &paths)
	return paths
}

// collectDiscoveryPaths recursively collects discovery paths for wildcards
func (t *pathTree) collectDiscoveryPaths(node *pathNode, currentPath string, paths *[]string) {
	// Build the current path
	if node.segment != "" {
		if currentPath != "" {
			currentPath += "."
		}
		currentPath += node.segment
	}

	// If this is a wildcard, we need to discover at this level
	if node.isWildcard {
		// The discovery path is everything before the wildcard, with a trailing dot
		discoveryPath := ""
		segments := strings.Split(currentPath, ".")
		for i := 0; i < len(segments)-1; i++ {
			if i > 0 {
				discoveryPath += "."
			}
			discoveryPath += segments[i]
		}
		if discoveryPath != "" {
			discoveryPath += "."
		}
		// Only add if not already present
		found := false
		for _, p := range *paths {
			if p == discoveryPath {
				found = true
				break
			}
		}
		if !found {
			*paths = append(*paths, discoveryPath)
		}
		// Don't recurse further - we need to resolve this wildcard first
		return
	}

	// Recurse to children
	for _, child := range node.children {
		t.collectDiscoveryPaths(child, currentPath, paths)
	}
}

// getNextLevelPaths generates discovery paths for the next wildcard level
// based on discovered indices at the current level
func (t *pathTree) getNextLevelPaths(discoveryPath string, indices []int) []string {
	if len(indices) == 0 {
		return nil
	}

	var nextPaths []string
	pathWithoutDot := strings.TrimSuffix(discoveryPath, ".")

	// For each index, build the expanded path and find next wildcards
	for _, idx := range indices {
		expandedPath := pathWithoutDot + "." + strconv.Itoa(idx)

		// Find the next wildcard level from this expanded path
		nextWildcard := t.findNextWildcard(expandedPath)
		if nextWildcard != "" {
			// Each index gets its own discovery path
			nextPaths = append(nextPaths, nextWildcard)
		}
	}

	return nextPaths
}

// findNextWildcard finds the next discovery path after the given expanded path
func (t *pathTree) findNextWildcard(expandedPath string) string {
	// We need to traverse the tree following the expanded path and find the next wildcard
	segments := strings.Split(expandedPath, ".")
	current := t.root

	// First, navigate to where we are in the tree
	// We need to match indices with wildcards
	for _, segment := range segments {
		if current.children == nil {
			return ""
		}

		found := false
		// Try exact match first
		if child, exists := current.children[segment]; exists {
			current = child
			found = true
		} else {
			// Check if this is a number that should match a wildcard
			if _, err := strconv.Atoi(segment); err == nil {
				if wildcardChild, exists := current.children["*"]; exists {
					current = wildcardChild
					found = true
				}
			}
		}

		if !found {
			return ""
		}
	}

	// Now look for the next wildcard in the subtree
	// Pass the expanded path so it includes the actual indices
	return t.findNextWildcardFrom(current, expandedPath)
}

// findNextWildcardFrom finds the next wildcard path from a given node
func (t *pathTree) findNextWildcardFrom(node *pathNode, basePath string) string {
	// Look through children to find the path to the next wildcard
	for segment, child := range node.children {
		// Skip wildcard at this level - we're looking for concrete paths
		if segment == "*" {
			continue
		}

		// This is a concrete segment (like "WLANConfiguration")
		// Build the path including this segment
		nextPath := basePath + "." + segment

		// Check if this child has a wildcard child
		if _, hasWildcard := child.children["*"]; hasWildcard {
			// Found the next wildcard level!
			// Return the discovery path for this level
			return nextPath + "."
		}

		// If no immediate wildcard, search deeper
		if !child.isLeaf {
			result := t.findNextWildcardFrom(child, nextPath)
			if result != "" {
				return result
			}
		}
	}

	// Check if there's a wildcard at this immediate level
	if _, exists := node.children["*"]; exists {
		// This means we have a wildcard right here
		// This shouldn't happen if we properly expanded the previous level
		return basePath + "."
	}

	return ""
}

// generateExpandedPaths generates all fully expanded paths using the cache
func (t *pathTree) generateExpandedPaths(cache map[string][]int) []string {
	if t.root == nil {
		return nil
	}

	var paths []string
	t.expandPaths(t.root, "", cache, &paths)
	return paths
}

// expandPaths recursively expands paths in the tree using cached indices
func (t *pathTree) expandPaths(node *pathNode, currentPath string, cache map[string][]int, result *[]string) {
	// Handle the root node
	if node.segment == "" && node == t.root {
		// Start expansion from children
		for _, child := range node.children {
			t.expandPaths(child, "", cache, result)
		}
		return
	}

	// Handle wildcard nodes
	if node.isWildcard {
		// Get the discovery path (parent path with trailing dot)
		discoveryPath := currentPath
		if currentPath != "" {
			discoveryPath += "."
		}

		// Look up indices in cache
		indices, exists := cache[discoveryPath]
		if !exists || len(indices) == 0 {
			// No indices found, can't expand this branch
			return
		}

		// Expand for each index
		for _, idx := range indices {
			indexPath := currentPath
			if indexPath != "" {
				indexPath += "."
			}
			indexPath += strconv.Itoa(idx)

			// Continue with children
			for _, child := range node.children {
				t.expandPaths(child, indexPath, cache, result)
			}
		}
		return
	}

	// Handle regular nodes
	if currentPath != "" {
		currentPath += "."
	}
	currentPath += node.segment

	// If this is a leaf, add to results
	if node.isLeaf {
		*result = append(*result, currentPath)
		return
	}

	// Continue with children
	for _, child := range node.children {
		t.expandPaths(child, currentPath, cache, result)
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

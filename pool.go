package expander

import "sync"

// expanderPool manages a pool of wildcard expanders for performance optimization.
var expanderPool = sync.Pool{
	New: func() any {
		return &wildcardExpander{
			discoveredPaths:      make(map[string][]int),
			pendingPaths:         make([]string, 0, 8),
			completedPaths:       make([]string, 0, 16),
			expectedFinalPaths:   make(map[string]bool),
			registeredFinalPaths: make(map[string]bool),
		}
	},
}

// New creates a new expander from the pool, initialized with the given wildcard path.
// The returned expander should be released back to the pool using Release() when done.
func New(wildcardPath string) (Expander, error) {
	exp := expanderPool.Get().(*wildcardExpander)

	if err := exp.reset(wildcardPath); err != nil {
		expanderPool.Put(exp)
		return nil, err
	}

	return exp, nil
}

// Release returns an expander to the pool for reuse.
// The expander should not be used after calling Release().
func Release(exp Expander) {
	if wExp, ok := exp.(*wildcardExpander); ok {
		expanderPool.Put(wExp)
	}
}

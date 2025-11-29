package expander

import "sync"

// expanderPool manages a pool of expanders for performance optimization.
// When an expander is retrieved from the pool, it starts with a fresh state.
var expanderPool = sync.Pool{
	New: func() any {
		return &Expander{
			paths: pathTree{
				root: &pathNode{
					children: make(map[string]*pathNode),
				},
			},
			cache:                make(map[string][]int),
			processedDiscoveries: make(map[string]bool),
			expandedSet:          make(map[string]bool),
			pendingDiscoveries:   make([]string, 0, 8),
			expandedPaths:        make([]string, 0, 16),
		}
	},
}

// Get retrieves an expander from the pool with a fresh state.
// The expander should be returned to the pool using Release() when done.
// If you want to reuse the cache, keep the expander instance and don't release it.
func Get() *Expander {
	exp := expanderPool.Get().(*Expander)
	// Ensure clean state
	exp.Reset()
	return exp
}

// Release returns an expander to the pool for reuse.
// The expander's state will be reset when it's retrieved again.
// Do not use the expander after calling Release().
func Release(exp *Expander) {
	if exp != nil {
		expanderPool.Put(exp)
	}
}

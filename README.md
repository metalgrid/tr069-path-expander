# TR-069 Wildcard Expansion Library

A high-performance Go library for expanding TR-069 parameter paths containing wildcards (`*`) through iterative discovery.

## Overview

This library manages the complex process of expanding TR-069 parameter paths that contain wildcards by maintaining internal state across multiple discovery iterations. It's designed for high-performance scenarios where expander instances are frequently created and destroyed.

## Features

- **Stateful Expansion**: Maintains internal state across discovery iterations
- **Multi-level Wildcards**: Supports multiple wildcards in a single parameter path
- **Performance Optimized**: Uses `sync.Pool` for object reuse to reduce GC pressure
- **Simple API**: Clean, intuitive interface following Go idioms
- **Single-threaded**: Optimized for single-worker usage (no thread safety overhead)

## Installation

```bash
go get github.com/your-org/tr069-expander
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/your-org/tr069-expander"
)

func main() {
    // Get an expander from the pool
    exp := expander.New("InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable")
    defer expander.Release(exp) // Return to pool when done
    
    // Discovery iteration 1: Get first discovery path
    path, hasMore := exp.NextDiscoveryPath()
    fmt.Println(path) // "InternetGatewayDevice.LANDevice."
    
    // Register discovered indices (simulating external discovery)
    exp.RegisterIndices("InternetGatewayDevice.LANDevice", []int{1, 2, 3})
    
    // Discovery iteration 2: Get next level paths
    for {
        path, hasMore := exp.NextDiscoveryPath()
        if !hasMore {
            break
        }
        fmt.Println(path) // "InternetGatewayDevice.LANDevice.1.WLANConfiguration."
                         // "InternetGatewayDevice.LANDevice.2.WLANConfiguration."
                         // "InternetGatewayDevice.LANDevice.3.WLANConfiguration."
        
        // Register indices for each path (simulating external discovery)
        if path == "InternetGatewayDevice.LANDevice.1.WLANConfiguration." {
            exp.RegisterIndices("InternetGatewayDevice.LANDevice.1.WLANConfiguration", []int{1, 2, 3})
        } else {
            exp.RegisterIndices(path[:len(path)-1], []int{}) // Empty results
        }
    }
    
    // Get final expanded paths
    expanded := exp.ExpandedPaths()
    for _, path := range expanded {
        fmt.Println(path)
        // Output:
        // InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable
        // InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Enable
        // InternetGatewayDevice.LANDevice.1.WLANConfiguration.3.Enable
    }
}
```

## API Reference

### Core Interface

```go
type Expander interface {
    // NextDiscoveryPath returns the next path segment for discovery.
    // Returns (path, true) if there's a path to discover, ("", false) if complete.
    NextDiscoveryPath() (string, bool)
    
    // RegisterIndices registers discovered indices for a given path.
    // Path should match the path returned by NextDiscoveryPath (without trailing dot).
    RegisterIndices(path string, indices []int) error
    
    // IsComplete returns true when all wildcards have been expanded.
    IsComplete() bool
    
    // ExpandedPaths returns all fully expanded parameter paths.
    // Only valid when IsComplete() returns true.
    ExpandedPaths() []string
}
```

### Pool Management

```go
// New creates a new expander from the pool, initialized with the given wildcard path.
func New(wildcardPath string) Expander

// Release returns an expander to the pool for reuse.
func Release(exp Expander)
```

## Usage Patterns

### Basic Single Wildcard

```go
exp := expander.New("Device.WiFi.AccessPoint.*.Enable")
defer expander.Release(exp)

// First discovery
path, _ := exp.NextDiscoveryPath() // "Device.WiFi.AccessPoint."
exp.RegisterIndices("Device.WiFi.AccessPoint", []int{1, 2})

// Get results
if exp.IsComplete() {
    paths := exp.ExpandedPaths()
    // ["Device.WiFi.AccessPoint.1.Enable", "Device.WiFi.AccessPoint.2.Enable"]
}
```

### Multi-level Wildcards

```go
exp := expander.New("Device.IP.Interface.*.IPv4Address.*.IPAddress")
defer expander.Release(exp)

// Discovery happens in levels
for !exp.IsComplete() {
    path, hasMore := exp.NextDiscoveryPath()
    if !hasMore {
        break
    }
    
    // Perform external discovery for 'path'
    indices := performDiscovery(path)
    exp.RegisterIndices(path[:len(path)-1], indices)
}

// Get all expanded paths
expanded := exp.ExpandedPaths()
```

### Error Handling

```go
exp := expander.New("Device.WiFi.AccessPoint.*.Enable")
defer expander.Release(exp)

path, _ := exp.NextDiscoveryPath()

// Register indices with validation
if err := exp.RegisterIndices("InvalidPath", []int{1, 2}); err != nil {
    log.Printf("Failed to register indices: %v", err)
}
```

## Performance Considerations

### Object Pooling

The library uses `sync.Pool` to reuse expander instances, significantly reducing GC pressure in high-throughput scenarios:

```go
// Good: Uses pool
exp := expander.New(path)
defer expander.Release(exp)

// Bad: Creates new instance each time
exp := &wildcardExpander{path: path} // Don't do this
```

### Memory Efficiency

- Pre-allocates slices where possible
- Reuses string builders for path construction
- Minimizes allocations in hot paths

### Benchmarks

```bash
go test -bench=. -benchmem
```

Expected performance:
- Single wildcard expansion: ~500ns/op, 0 allocs
- Multi-level expansion: ~2μs/op, minimal allocs
- Pool get/release: ~50ns/op, 0 allocs

## Advanced Usage

### Custom Discovery Logic

```go
type DiscoveryClient struct {
    // Your TR-069 client implementation
}

func (c *DiscoveryClient) ExpandPath(wildcardPath string) ([]string, error) {
    exp := expander.New(wildcardPath)
    defer expander.Release(exp)
    
    for !exp.IsComplete() {
        path, hasMore := exp.NextDiscoveryPath()
        if !hasMore {
            break
        }
        
        // Use your TR-069 client to discover indices
        indices, err := c.GetParameterNames(path)
        if err != nil {
            return nil, err
        }
        
        if err := exp.RegisterIndices(path[:len(path)-1], indices); err != nil {
            return nil, err
        }
    }
    
    return exp.ExpandedPaths(), nil
}
```

### Batch Processing

```go
func expandMultiplePaths(paths []string) map[string][]string {
    results := make(map[string][]string)
    
    for _, path := range paths {
        exp := expander.New(path)
        
        // Perform expansion...
        
        results[path] = exp.ExpandedPaths()
        expander.Release(exp)
    }
    
    return results
}
```

## Error Conditions

The library handles several error conditions gracefully:

- **Invalid wildcard paths**: Paths without wildcards return themselves
- **Mismatched path registration**: Returns error when registered path doesn't match expected
- **Empty index responses**: Handled correctly, results in no expanded paths for that branch
- **Malformed paths**: Validated during initialization

## Testing

The library includes comprehensive tests using Ginkgo and Gomega:

```bash
go test ./...
```

Test coverage includes:
- Single and multi-level wildcard expansion
- Edge cases (empty indices, invalid paths)
- Performance benchmarks
- Pool management
- Error conditions

## Contributing

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Implement the feature
5. Ensure all tests pass
6. Submit a pull request

## License

MIT License - see LICENSE file for details.
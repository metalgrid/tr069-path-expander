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
go get github.com/metalgrid/tr069-path-expander
```

## Quick Start

```go
type Expander interface {
    // NextDiscoveryPath returns the next path segment for discovery
    NextDiscoveryPath() (string, bool)
    
    // RegisterParameterNames registers discovered parameter names for a given path
    // Automatically extracts indices from the parameter names
    RegisterParameterNames(path string, parameterNames []string) error
    
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
parameterNames := []string{"Device.WiFi.AccessPoint.1", "Device.WiFi.AccessPoint.2"}
exp.RegisterParameterNames("Device.WiFi.AccessPoint", parameterNames)

// Get results
if exp.IsComplete() {
    paths := exp.ExpandedPaths()
    // ["Device.WiFi.AccessPoint.1.Enable", "Device.WiFi.AccessPoint.2.Enable"]
}
```

### Multi-level Wildcards

The library provides robust support for multi-level wildcard expansion, including edge cases where some intermediate paths return empty indices:

```go
exp := expander.New("InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable")
defer expander.Release(exp)

// Step 1: Get first discovery path
path, _ := exp.NextDiscoveryPath()
// Returns: "InternetGatewayDevice.LANDevice."

// Step 2: Register first level parameter names
firstLevelParams := []string{
    "InternetGatewayDevice.LANDevice.1",
    "InternetGatewayDevice.LANDevice.2",
    "InternetGatewayDevice.LANDevice.3",
}
exp.RegisterParameterNames("InternetGatewayDevice.LANDevice", firstLevelParams)

// Step 3: Get second level discovery paths
var secondLevelPaths []string
for {
    path, hasMore := exp.NextDiscoveryPath()
    if !hasMore {
        break
    }
    secondLevelPaths = append(secondLevelPaths, path)
}
// Returns: ["InternetGatewayDevice.LANDevice.1.WLANConfiguration.",
//           "InternetGatewayDevice.LANDevice.2.WLANConfiguration.",
//           "InternetGatewayDevice.LANDevice.3.WLANConfiguration."]

// Step 4: Register second level parameter names (including empty responses)
secondLevelParams1 := []string{
    "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1",
    "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2",
    "InternetGatewayDevice.LANDevice.1.WLANConfiguration.3",
}
exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.1.WLANConfiguration", secondLevelParams1)
exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.2.WLANConfiguration", []string{})      // Empty response
exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.3.WLANConfiguration", []string{})      // Empty response

// Step 5: Expansion is complete when all expected registrations are received
if exp.IsComplete() {
    expanded := exp.ExpandedPaths()
    // Returns: ["InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable",
    //           "InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Enable",
    //           "InternetGatewayDevice.LANDevice.1.WLANConfiguration.3.Enable"]
}
```

#### Edge Case Handling

The library correctly handles common TR-069 scenarios where:
- Some device instances exist but have no sub-instances (empty indices)
- All final-level paths must be registered before completion
- Mixed empty and non-empty index registrations in the same expansion

### Error Handling

```go
exp := expander.New("Device.WiFi.AccessPoint.*.Enable")
defer expander.Release(exp)

path, _ := exp.NextDiscoveryPath()

// Register parameter names with validation
invalidParams := []string{"InvalidPath.1", "InvalidPath.2"}
if err := exp.RegisterParameterNames("InvalidPath", invalidParams); err != nil {
    log.Printf("Failed to register parameter names: %v", err)
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
- Single wildcard expansion: ~773ns/op, 19 allocs
- Multi-level expansion: ~1.8μs/op, 37 allocs
- Pool get/release: ~125ns/op, 4 allocs

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
        
        if err := exp.RegisterParameterNames(path[:len(path)-1], indices); err != nil {
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

## External Project Integration

### Adding to Your Project

1. **Install the library**:
   ```bash
   go get github.com/metalgrid/tr069-path-expander
   ```

2. **Import in your Go code**:
   ```go
        import expander "github.com/metalgrid/tr069-path-expander"   ```

3. **Basic integration example**:
   ```go
   package main

   import (
       "fmt"
       "log"
       
       expander "github.com/metalgrid/tr069-path-expander"
   )

   func main() {
       // Create expander for TR-069 parameter path
       exp, err := expander.New("Device.WiFi.AccessPoint.*.Enable")
       if err != nil {
           log.Fatal(err)
       }
       defer expander.Release(exp)

       // Integrate with your TR-069 discovery logic
       for !exp.IsComplete() {
           path, hasMore := exp.NextDiscoveryPath()
           if !hasMore {
               break
           }

    // Call your TR-069 client to discover parameter names
    parameterNames := discoverParameterNames(path) // Your implementation
    
    // Register the discovered parameter names
    pathWithoutDot := path[:len(path)-1]
    if err := exp.RegisterParameterNames(pathWithoutDot, parameterNames); err != nil {
        log.Printf("Failed to register parameter names: %v", err)
        continue
    }       }

       // Get all expanded parameter paths
       expandedPaths := exp.ExpandedPaths()
       fmt.Printf("Expanded paths: %v\n", expandedPaths)
   }

// Your TR-069 discovery implementation
func discoverParameterNames(path string) []string {
    // Implement your TR-069 GetParameterNames call here
    // Return the actual parameter names from the response
    return []string{
        "Device.WiFi.AccessPoint.1",
        "Device.WiFi.AccessPoint.2", 
        "Device.WiFi.AccessPoint.3",
    } // Example
}   ```

### Integration with Popular TR-069 Libraries

#### With go-cwmp
```go
import (
    "github.com/your-org/go-cwmp"
    expander "github.com/metalgrid/tr069-path-expander"
)

func expandWithCWMP(client *cwmp.Client, wildcardPath string) ([]string, error) {
    exp, err := expander.New(wildcardPath)
    if err != nil {
        return nil, err
    }
    defer expander.Release(exp)

    for !exp.IsComplete() {
        path, hasMore := exp.NextDiscoveryPath()
        if !hasMore {
            break
        }

        // Use CWMP client to discover parameter names
        params, err := client.GetParameterNames(path, false)
        if err != nil {
            return nil, err
        }

        // Use parameter names directly (no need to extract indices)
        pathWithoutDot := path[:len(path)-1]
        if err := exp.RegisterParameterNames(pathWithoutDot, params); err != nil {
            return nil, err
        }
    }

    return exp.ExpandedPaths(), nil
}
```

### Module Requirements

- **Go version**: 1.21 or later
- **Dependencies**: None (only test dependencies for development)
- **Import path**: `github.com/metalgrid/tr069-path-expander`

### Versioning

The library follows semantic versioning (SemVer):
- **v1.x.x**: Stable API, backward compatible
- **v0.x.x**: Development versions, API may change

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
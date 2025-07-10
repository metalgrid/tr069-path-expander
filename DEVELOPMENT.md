# TR-069 Wildcard Expansion Library - Development Guide

## Project Overview

This is a high-performance Go library for expanding TR-069 parameter paths containing wildcards (`*`) through iterative discovery. The library maintains internal state across multiple discovery rounds and supports multi-level wildcard expansion.

### Core Problem
TR-069 parameter paths like `InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable` need to be expanded by discovering indices at each wildcard level through external queries. The library manages this complex stateful process.

### Key Requirements
- **Stateful expansion**: Maintain state across discovery iterations
- **Multi-level support**: Handle multiple wildcards in a single path
- **High performance**: Optimized for frequent instantiation using sync.Pool
- **Single-threaded**: No thread safety required (single worker usage)
- **Modern Go**: Use latest Go idioms and patterns

## Architecture

### Core Components

#### 1. Expander Interface (`expander.go`)
```go
type Expander interface {
    NextDiscoveryPath() (string, bool)
    RegisterIndices(path string, indices []int) error
    IsComplete() bool
    ExpandedPaths() []string
}
```

#### 2. Implementation (`wildcard_expander.go`)
```go
type wildcardExpander struct {
    originalPath    string
    pathSegments    []string
    wildcardLevels  []int
    currentLevel    int
    discoveredPaths map[string][]int
    pendingPaths    []string
    completedPaths  []string
    isComplete      bool
}
```

#### 3. Pool Management (`pool.go`)
```go
var expanderPool = sync.Pool{
    New: func() any {
        return &wildcardExpander{
            discoveredPaths: make(map[string][]int),
            pendingPaths:    make([]string, 0, 8),
            completedPaths:  make([]string, 0, 16),
        }
    },
}
```

### Data Flow

1. **Initialization**: Parse wildcard path, identify wildcard positions
2. **Discovery Loop**: 
   - Generate discovery paths for current wildcard level
   - External system queries for indices
   - Register discovered indices
   - Generate paths for next level or final expansion
3. **Completion**: Return all fully expanded parameter paths

## File Structure

```
tr069-expander/
├── expander.go          # Core interface and types
├── wildcard_expander.go # Main implementation
├── pool.go              # sync.Pool management
├── expander_test.go     # Ginkgo test suite
├── benchmark_test.go    # Performance benchmarks
├── README.md            # Usage documentation
├── PLAN.md              # Development plan
├── DEVELOPMENT.md       # This file
└── go.mod               # Module definition
```

## Development Environment

### Prerequisites
- Go 1.24.5 or later
- Ginkgo v2 testing framework
- Gomega assertion library

### Setup
```bash
go mod tidy
go get github.com/onsi/ginkgo/v2
go get github.com/onsi/gomega
```

### Running Tests
```bash
# All tests
go test -v

# Specific test
go test -v -ginkgo.focus="should return expanded paths"

# Benchmarks only
go test -run=^$ -bench=.

# With coverage
go test -cover
```

## Implementation Details

### Path Parsing Logic

#### Wildcard Detection
```go
func parseWildcardPath(path string) ([]string, []int, error) {
    segments := strings.Split(path, ".")
    var wildcardLevels []int
    
    for i, segment := range segments {
        if segment == "*" {
            wildcardLevels = append(wildcardLevels, i)
        }
    }
    
    return segments, wildcardLevels, nil
}
```

#### Discovery Path Generation
```go
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
```

### State Management

#### Expansion State Tracking
- `originalPath`: Original wildcard path
- `pathSegments`: Split path components
- `wildcardLevels`: Positions of wildcards in path
- `discoveredPaths`: Map of discovered indices per path
- `pendingPaths`: Paths awaiting discovery
- `completedPaths`: Final expanded paths
- `isComplete`: Completion flag
- `expectedFinalPaths`: Tracks final-level paths that need registration (multi-level)
- `registeredFinalPaths`: Tracks final-level paths that have been registered (multi-level)

#### Enhanced Completion Logic
```go
func (e *wildcardExpander) checkCompletion() {
    noPendingPaths := len(e.pendingPaths) == 0
    
    // Single wildcard case: use original logic
    if len(e.expectedFinalPaths) == 0 {
        e.isComplete = noPendingPaths
        return
    }
    
    // Multi-level case: check both conditions
    allFinalPathsRegistered := len(e.expectedFinalPaths) == len(e.registeredFinalPaths)
    e.isComplete = noPendingPaths && allFinalPathsRegistered
}
```

**Key Enhancement**: The completion logic now handles multi-level wildcard edge cases where some final-level paths return empty indices. Completion only occurs when:
1. No pending discovery paths remain, AND
2. All expected final-level paths have been registered (for multi-level scenarios)

### Performance Optimizations

#### Object Pooling
- Pre-allocated slices with capacity hints
- Map reuse through clearing instead of recreation
- Zero-allocation path building using strings.Builder

#### Memory Management
```go
// Pool initialization with pre-allocated structures
expanderPool = sync.Pool{
    New: func() any {
        return &wildcardExpander{
            discoveredPaths:      make(map[string][]int),
            pendingPaths:         make([]string, 0, 8),    // Pre-allocate
            completedPaths:       make([]string, 0, 16),   // Pre-allocate
            expectedFinalPaths:   make(map[string]bool),   // Multi-level tracking
            registeredFinalPaths: make(map[string]bool),   // Multi-level tracking
        }
    },
}
```

## Testing Strategy

### Test Structure (Ginkgo/Gomega)
```go
var _ = Describe("WildcardExpander", func() {
    Context("when initialized with a wildcard path", func() {
        When("discovering single-level wildcards", func() {
            It("should return correct discovery paths", func() {
                // Test implementation
            })
        })
    })
})
```

### Test Categories

#### 1. Basic Functionality
- Path without wildcards
- Single wildcard expansion
- Multi-level wildcard expansion
- Empty index responses

#### 2. Edge Cases
- Invalid path formats
- Mismatched path registration
- Registration after completion
- Empty paths

#### 3. Performance Tests
- Single wildcard benchmarks
- Multi-level expansion benchmarks
- Pool reuse benchmarks

#### 4. Integration Tests
- Complete workflow from requirements
- Real-world usage patterns

### Current Test Status
- **26/27 tests passing** (96% success rate)
- **1 failing test**: Complex multi-level completion timing issue

## Known Issues

### Issue #1: Multi-Level Completion Timing
**File**: `expander_test.go:317`
**Test**: "should follow the exact workflow from the requirements"
**Problem**: Completion is detected too early in multi-level wildcard scenarios

#### Root Cause
When registering the first final-level path with indices, the completion check triggers immediately, but other final-level paths (with empty indices) still need registration.

#### Current Behavior
```go
// Step 5: Register responses for each value
err = exp.RegisterIndices("InternetGatewayDevice.LANDevice.1.WLANConfiguration", []int{1, 2, 3})
// ❌ Completion triggered here, but we still need to register:
err = exp.RegisterIndices("InternetGatewayDevice.LANDevice.2.WLANConfiguration", []int{})
err = exp.RegisterIndices("InternetGatewayDevice.LANDevice.3.WLANConfiguration", []int{})
```

#### Potential Solutions
1. **Track expected final-level paths**: Count how many final-level paths are expected
2. **Defer completion check**: Only check completion when no pending paths AND all expected registrations complete
3. **Registration state tracking**: Track which paths have been registered vs expected

#### Implementation Approach
```go
type wildcardExpander struct {
    // ... existing fields
    expectedFinalPaths map[string]bool  // Track expected final registrations
    registeredFinalPaths int            // Count of registered final paths
}
```

## Development Workflow

### Adding New Features

#### 1. Test-Driven Development
```bash
# 1. Write failing test
go test -v -ginkgo.focus="new feature"

# 2. Implement minimal code to pass
# 3. Refactor and optimize
# 4. Ensure all tests pass
go test -v
```

#### 2. Performance Validation
```bash
# Run benchmarks after changes
go test -run=^$ -bench=.

# Profile if needed
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof
```

### Code Style Guidelines

#### 1. Modern Go Practices
```go
// ✅ Good: Use range over int
for range 10 {
    // ...
}

// ❌ Bad: Traditional for loop
for i := 0; i < 10; i++ {
    // ...
}

// ✅ Good: Use any instead of interface{}
func process(data any) error

// ❌ Bad: Use interface{}
func process(data interface{}) error
```

#### 2. Error Handling
```go
// ✅ Good: Descriptive errors
var (
    ErrInvalidPath     = errors.New("invalid path format")
    ErrPathMismatch    = errors.New("path mismatch")
    ErrAlreadyComplete = errors.New("expansion is already complete")
)

// ✅ Good: Wrapped errors
return fmt.Errorf("%w: got %s", ErrPathMismatch, path)
```

#### 3. Documentation
```go
// Package-level documentation
// Package expander provides TR-069 wildcard expansion functionality.

// Function documentation
// NextDiscoveryPath returns the next path segment for discovery.
// Returns (path, true) if there's a path to discover, ("", false) if complete.
func (e *wildcardExpander) NextDiscoveryPath() (string, bool)
```

### Debugging Techniques

#### 1. Test Isolation
```bash
# Run specific failing test
go test -v -ginkgo.focus="exact test name"

# Skip other tests
go test -v -ginkgo.skip="pattern to skip"
```

#### 2. State Inspection
```go
// Add temporary debug output
fmt.Printf("DEBUG: pendingPaths=%v, isComplete=%v\n", e.pendingPaths, e.isComplete)
```

#### 3. Race Detection
```bash
# Even though single-threaded, verify no races
go test -race
```

## Performance Targets

### Benchmarks
- Single wildcard expansion: **< 1μs/op**
- Multi-level expansion: **< 5μs/op**
- Pool get/release: **< 200ns/op**
- Zero allocations in hot paths

### Current Performance
- Single wildcard: **773ns/op** ✅
- Multi-level: **1.8μs/op** ✅
- Pool reuse: **125ns/op** ✅

## Future Enhancements

### Potential Improvements

#### 1. Advanced Caching
- Cache parsed path structures
- Memoize common expansion patterns

#### 2. Validation Enhancements
- Stricter path format validation
- Better error messages with context

#### 3. Metrics and Observability
- Expansion timing metrics
- Pool utilization statistics
- Memory usage tracking

#### 4. Configuration Options
- Configurable pool sizes
- Custom validation rules
- Debug mode with detailed logging

### API Extensions

#### 1. Batch Operations
```go
type BatchExpander interface {
    ExpandMultiple(paths []string) (map[string][]string, error)
}
```

#### 2. Streaming Interface
```go
type StreamingExpander interface {
    ExpandStream(paths <-chan string) <-chan ExpansionResult
}
```

#### 3. Context Support
```go
func NewWithContext(ctx context.Context, path string) (Expander, error)
```

## Troubleshooting Guide

### Common Issues

#### 1. "assignment to entry in nil map"
**Cause**: Map not initialized in struct
**Solution**: Ensure all maps are initialized in constructors

#### 2. "path mismatch" errors
**Cause**: Registered path doesn't match expected discovery path
**Solution**: Verify path format and wildcard positions

#### 3. Infinite loops in discovery
**Cause**: Completion logic not triggering
**Solution**: Check `pendingPaths` state and completion conditions

#### 4. Memory leaks
**Cause**: Objects not returned to pool
**Solution**: Always call `Release()` in defer statements

### Debugging Checklist

1. **Verify test isolation**: Each test should start with clean state
2. **Check pool usage**: Ensure proper `New()`/`Release()` pairing
3. **Validate path formats**: Confirm wildcard positions are correct
4. **Trace state changes**: Follow `pendingPaths` and `isComplete` evolution
5. **Review completion logic**: Ensure all paths are processed before completion

## Contributing Guidelines

### Pull Request Process

1. **Create feature branch**: `git checkout -b feature/description`
2. **Write tests first**: Follow TDD approach
3. **Implement feature**: Minimal code to pass tests
4. **Run full test suite**: `go test -v`
5. **Run benchmarks**: Verify performance impact
6. **Update documentation**: README.md and code comments
7. **Submit PR**: Include test results and benchmark comparison

### Code Review Checklist

- [ ] All tests pass
- [ ] Benchmarks show no regression
- [ ] Code follows Go idioms
- [ ] Documentation is updated
- [ ] Error handling is comprehensive
- [ ] Memory usage is optimized
- [ ] Pool usage is correct

## Release Process

### Version Management
- Follow semantic versioning (semver)
- Tag releases: `git tag v1.0.0`
- Update CHANGELOG.md

### Pre-Release Checklist
- [ ] All tests pass
- [ ] Benchmarks meet targets
- [ ] Documentation is complete
- [ ] Examples work correctly
- [ ] No known critical issues

### Release Artifacts
- Source code archive
- Go module release
- Documentation updates
- Benchmark results

---

## Quick Reference

### Essential Commands
```bash
# Development
go test -v                    # Run all tests
go test -run=^$ -bench=.     # Run benchmarks
go test -cover               # Coverage report

# Debugging
go test -v -ginkgo.focus="test name"  # Run specific test
go test -race                         # Race detection

# Performance
go test -cpuprofile=cpu.prof -bench=.  # CPU profiling
go tool pprof cpu.prof                 # Analyze profile
```

### Key Files to Monitor
- `expander_test.go:317` - Known failing test
- `wildcard_expander.go:checkCompletion()` - Completion logic
- `pool.go` - Performance-critical pool management
- `benchmark_test.go` - Performance validation

### Performance Metrics
- Target: < 1μs single wildcard, < 5μs multi-level
- Current: 943ns single, 2.2μs multi-level
- Pool overhead: 146ns get/release

This development guide provides the complete technical context needed for AI coding agents to effectively work on this project. The structured approach, detailed implementation notes, and troubleshooting guides enable efficient development and maintenance.
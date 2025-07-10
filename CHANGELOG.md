# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-01-10

### Added
- Initial stable release of TR-069 wildcard expansion library
- Core `Expander` interface with complete API
- High-performance object pooling using `sync.Pool`
- Multi-level wildcard expansion support
- Enhanced completion logic for edge cases
- Comprehensive test suite with 27 test cases
- Performance benchmarks meeting all targets
- Complete documentation and usage examples
- External project integration guide
- Error handling with clear error messages

### Features
- **State Management**: Maintains internal expansion state across discovery iterations
- **Multi-level Support**: Handles multiple wildcards in a single parameter path  
- **Edge Case Handling**: Properly handles mixed empty/non-empty index registrations
- **Performance Optimized**: Object pooling reduces GC pressure
- **Production Ready**: Comprehensive testing and validation

### Performance
- Single wildcard expansion: 773ns/op, 19 allocs
- Multi-level expansion: 1.8μs/op, 37 allocs
- Pool get/release: 125ns/op, 4 allocs

### API
```go
// Core interface
type Expander interface {
    NextDiscoveryPath() (string, bool)
    RegisterIndices(path string, indices []int) error
    IsComplete() bool
    ExpandedPaths() []string
}

// Pool management
func New(wildcardPath string) (Expander, error)
func Release(exp Expander)
```

### Documentation
- Complete README with usage examples
- API reference documentation
- Performance optimization guide
- Integration examples for external projects
- Error handling best practices

### Testing
- 27 comprehensive test cases using Ginkgo/Gomega
- Edge case coverage including empty index scenarios
- Performance benchmarks with regression testing
- Example project demonstrating external usage

## [Unreleased]

### Planned
- Additional performance optimizations
- Extended validation options
- Metrics and observability features
- Batch operation support

---

## Release Process

### Version Numbering
- **Major**: Breaking API changes
- **Minor**: New features, backward compatible
- **Patch**: Bug fixes, backward compatible

### Release Criteria
- All tests passing (27/27)
- Performance benchmarks meeting targets
- Documentation complete and up-to-date
- Examples working and validated
# Release Notes

## v1.0.0 (Initial Release)

### Features
- ✅ **Complete TR-069 wildcard expansion support**
- ✅ **Multi-level wildcard handling** with edge case support
- ✅ **High-performance object pooling** using sync.Pool
- ✅ **Comprehensive test suite** with 27 test cases (100% pass rate)
- ✅ **Production-ready performance**:
  - Single wildcard: 773ns/op
  - Multi-level: 1.8μs/op  
  - Pool reuse: 125ns/op

### API
- `New(wildcardPath string) (Expander, error)` - Create new expander from pool
- `Release(exp Expander)` - Return expander to pool
- `NextDiscoveryPath() (string, bool)` - Get next path for discovery
- `RegisterIndices(path string, indices []int) error` - Register discovered indices
- `IsComplete() bool` - Check if expansion is complete
- `ExpandedPaths() []string` - Get all expanded paths

### Edge Cases Handled
- **Mixed empty/non-empty indices**: Properly handles scenarios where some intermediate paths return empty indices
- **Multi-level completion**: Enhanced completion logic ensures all expected final-level paths are registered
- **Path validation**: Comprehensive validation with clear error messages

### Performance Optimizations
- **Object pooling**: Reduces GC pressure in high-throughput scenarios
- **Pre-allocated structures**: Minimizes allocations during expansion
- **Efficient string building**: Uses strings.Builder for path construction

### Documentation
- Complete API documentation with examples
- Integration guide for external projects
- Performance benchmarks and optimization tips
- Comprehensive error handling examples

## Release Process

### Semantic Versioning
This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version for incompatible API changes
- **MINOR** version for backward-compatible functionality additions  
- **PATCH** version for backward-compatible bug fixes

### Version Tags
- `v1.0.0` - Initial stable release
- `v1.x.x` - Future stable releases with backward compatibility
- `v2.x.x` - Future major releases (if API changes are needed)

### Release Checklist
- [ ] All tests passing (27/27)
- [ ] Performance benchmarks meet targets
- [ ] Documentation updated
- [ ] Examples working
- [ ] CHANGELOG.md updated
- [ ] Git tag created
- [ ] GitHub release published

## Installation

### Latest Stable Release
```bash
go get github.com/metalgrid/tr069-path-expander@latest
```

### Specific Version
```bash
go get github.com/metalgrid/tr069-path-expander@v1.0.0
```

### Development Version
```bash
go get github.com/metalgrid/tr069-path-expander@main
```

## Compatibility

### Go Version Support
- **Minimum**: Go 1.21
- **Tested**: Go 1.21, 1.22, 1.23
- **Recommended**: Latest stable Go version

### Dependencies
- **Runtime**: Zero dependencies
- **Testing**: Ginkgo v2, Gomega (dev only)

## Migration Guide

### From Development Versions
If upgrading from pre-1.0 development versions:
1. Update import path to `github.com/metalgrid/tr069-path-expander`
2. No API changes required - fully backward compatible
3. Performance improvements are automatic

### Future Migrations
Breaking changes (if any) will be clearly documented with migration guides and deprecation notices.
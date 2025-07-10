package expander_test

import (
	"testing"

	expander "github.com/metalgrid/tr069-path-expander"
)

func BenchmarkSingleWildcard(b *testing.B) {
	for range b.N {
		exp, err := expander.New("Device.WiFi.AccessPoint.*.Enable")
		if err != nil {
			b.Fatal(err)
		}

		// Simulate discovery
		path, _ := exp.NextDiscoveryPath()
		pathWithoutDot := path[:len(path)-1]
		parameterNames := []string{
			pathWithoutDot + ".1",
			pathWithoutDot + ".2",
			pathWithoutDot + ".3",
		}
		exp.RegisterParameterNames(pathWithoutDot, parameterNames)

		// Get results
		if !exp.IsComplete() {
			b.Fatal("expansion not complete")
		}

		paths := exp.ExpandedPaths()
		if len(paths) != 3 {
			b.Fatalf("expected 3 paths, got %d", len(paths))
		}

		expander.Release(exp)
	}
}

func BenchmarkMultiWildcard(b *testing.B) {
	for range b.N {
		exp, err := expander.New("Device.IP.Interface.*.IPv4Address.*.IPAddress")
		if err != nil {
			b.Fatal(err)
		}

		// First level discovery
		path, _ := exp.NextDiscoveryPath()
		pathWithoutDot := path[:len(path)-1]
		firstLevelParams := []string{
			pathWithoutDot + ".1",
			pathWithoutDot + ".2",
		}
		exp.RegisterParameterNames(pathWithoutDot, firstLevelParams)

		// Second level discovery
		for {
			path, hasMore := exp.NextDiscoveryPath()
			if !hasMore {
				break
			}
			pathWithoutDot := path[:len(path)-1]
			// Simulate some interfaces having IPv4 addresses, others don't
			if path == "Device.IP.Interface.1.IPv4Address." {
				secondLevelParams := []string{
					pathWithoutDot + ".1",
					pathWithoutDot + ".2",
				}
				exp.RegisterParameterNames(pathWithoutDot, secondLevelParams)
			} else {
				// Empty response for interface 2
				exp.RegisterParameterNames(pathWithoutDot, []string{})
			}
		}

		// Get results
		if !exp.IsComplete() {
			b.Fatal("expansion not complete")
		}

		expander.Release(exp)
	}
}

func BenchmarkPoolReuse(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		for range b.N {
			exp, err := expander.New("Device.Test.*.Value")
			if err != nil {
				b.Fatal(err)
			}
			expander.Release(exp)
		}
	})
}

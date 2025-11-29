package expander_test

import (
	"testing"

	expander "github.com/metalgrid/tr069-path-expander/v2"
)

func BenchmarkSingleWildcard(b *testing.B) {
	for range b.N {
		exp := expander.Get()

		err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
		if err != nil {
			b.Fatal(err)
		}

		// Simulate discovery
		_, hasMore := exp.Next()
		if !hasMore {
			b.Fatal("expected discovery path")
		}

		err = exp.Register([]string{
			"Device.WiFi.AccessPoint.1",
			"Device.WiFi.AccessPoint.2",
			"Device.WiFi.AccessPoint.3",
		})
		if err != nil {
			b.Fatal(err)
		}

		// Get results
		_, hasMore = exp.Next()
		if hasMore {
			b.Fatal("expected completion")
		}

		paths, err := exp.Collect()
		if err != nil {
			b.Fatal(err)
		}
		if len(paths) != 3 {
			b.Fatalf("expected 3 paths, got %d", len(paths))
		}

		expander.Release(exp)
	}
}

func BenchmarkMultiWildcard(b *testing.B) {
	for range b.N {
		exp := expander.Get()

		err := exp.Add([]string{"Device.IP.Interface.*.IPv4Address.*.IPAddress"})
		if err != nil {
			b.Fatal(err)
		}

		// First level discovery
		_, hasMore := exp.Next()
		if !hasMore {
			b.Fatal("expected discovery path")
		}

		err = exp.Register([]string{
			"Device.IP.Interface.1",
			"Device.IP.Interface.2",
		})
		if err != nil {
			b.Fatal(err)
		}

		// Second level discovery
		for {
			path, hasMore := exp.Next()
			if !hasMore {
				break
			}

			// Simulate some interfaces having IPv4 addresses, others don't
			if path == "Device.IP.Interface.1.IPv4Address." {
				err = exp.Register([]string{
					"Device.IP.Interface.1.IPv4Address.1",
					"Device.IP.Interface.1.IPv4Address.2",
				})
			} else {
				// Empty response for interface 2
				err = exp.Register([]string{})
			}

			if err != nil {
				b.Fatal(err)
			}
		}

		// Get results
		_, err = exp.Collect()
		if err != nil {
			b.Fatal(err)
		}

		expander.Release(exp)
	}
}

func BenchmarkCommonAncestor(b *testing.B) {
	for range b.N {
		exp := expander.Get()

		// Add multiple paths with common ancestor
		err := exp.Add([]string{
			"Device.WiFi.AccessPoint.*.Enable",
			"Device.WiFi.AccessPoint.*.Status",
			"Device.WiFi.AccessPoint.*.Name",
			"Device.WiFi.AccessPoint.*.SSID",
		})
		if err != nil {
			b.Fatal(err)
		}

		// Should only need one discovery
		_, hasMore := exp.Next()
		if !hasMore {
			b.Fatal("expected discovery path")
		}

		err = exp.Register([]string{
			"Device.WiFi.AccessPoint.1",
			"Device.WiFi.AccessPoint.2",
			"Device.WiFi.AccessPoint.3",
		})
		if err != nil {
			b.Fatal(err)
		}

		// Should be complete
		_, hasMore = exp.Next()
		if hasMore {
			b.Fatal("expected completion")
		}

		paths, err := exp.Collect()
		if err != nil {
			b.Fatal(err)
		}
		if len(paths) != 12 { // 3 instances × 4 properties
			b.Fatalf("expected 12 paths, got %d", len(paths))
		}

		expander.Release(exp)
	}
}

func BenchmarkPoolReuse(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		for range b.N {
			exp := expander.Get()
			expander.Release(exp)
		}
	})
}

func BenchmarkDynamicAddition(b *testing.B) {
	for range b.N {
		exp := expander.Get()

		// Initial paths
		err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
		if err != nil {
			b.Fatal(err)
		}

		_, _ = exp.Next()
		err = exp.Register([]string{
			"Device.WiFi.AccessPoint.1",
			"Device.WiFi.AccessPoint.2",
		})
		if err != nil {
			b.Fatal(err)
		}

		// Dynamic addition (should use cache)
		err = exp.Add([]string{"Device.WiFi.AccessPoint.*.Status"})
		if err != nil {
			b.Fatal(err)
		}

		// Should not need new discovery
		_, hasMore := exp.Next()
		if hasMore {
			b.Fatal("expected no more discoveries")
		}

		paths, err := exp.Collect()
		if err != nil {
			b.Fatal(err)
		}
		if len(paths) != 4 { // 2 instances × 2 properties
			b.Fatalf("expected 4 paths, got %d", len(paths))
		}

		expander.Release(exp)
	}
}

package main

import (
	"fmt"
	"log"

	expander "github.com/metalgrid/tr069-path-expander/v2"
)

// simulateDiscovery simulates a TR-069 GetParameterNames call
func simulateDiscovery(path string) []string {
	// Simulate different discovery results based on path
	switch path {
	case "InternetGatewayDevice.LANDevice.":
		return []string{
			"InternetGatewayDevice.LANDevice.1",
			"InternetGatewayDevice.LANDevice.2",
			"InternetGatewayDevice.LANDevice.3",
		}
	case "InternetGatewayDevice.LANDevice.1.WLANConfiguration.":
		return []string{
			"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1",
			"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2",
			"InternetGatewayDevice.LANDevice.1.WLANConfiguration.3",
		}
	case "InternetGatewayDevice.LANDevice.2.WLANConfiguration.":
		return []string{} // Device 2 has no WLAN configurations
	case "InternetGatewayDevice.LANDevice.3.WLANConfiguration.":
		return []string{
			"InternetGatewayDevice.LANDevice.3.WLANConfiguration.1",
		}
	default:
		return []string{}
	}
}

func main() {
	fmt.Println("TR-069 Wildcard Expander v2 - Basic Usage Example")
	fmt.Println("==================================================")

	// Example 1: Single wildcard expansion
	fmt.Println("\n1. Single Wildcard Example:")
	singleWildcardExample()

	// Example 2: Multi-level wildcard expansion
	fmt.Println("\n2. Multi-level Wildcard Example:")
	multiLevelExample()

	// Example 3: Common ancestor optimization
	fmt.Println("\n3. Common Ancestor Optimization Example:")
	commonAncestorExample()

	// Example 4: Dynamic addition
	fmt.Println("\n4. Dynamic Addition Example:")
	dynamicAdditionExample()
}

func singleWildcardExample() {
	// Get expander from pool
	exp := expander.Get()
	defer expander.Release(exp)

	// Add path for expansion
	err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original path: Device.WiFi.AccessPoint.*.Enable\n")

	// Get discovery path
	path, hasMore := exp.Next()
	fmt.Printf("Discovery path: %s\n", path)

	if hasMore {
		// Simulate discovery - found 3 access points
		parameterNames := []string{
			"Device.WiFi.AccessPoint.1",
			"Device.WiFi.AccessPoint.2",
			"Device.WiFi.AccessPoint.3",
		}

		err := exp.Register(parameterNames)
		if err != nil {
			log.Printf("Error registering results: %v", err)
			return
		}

		fmt.Printf("Registered parameter names: %v\n", parameterNames)
	}

	// Check for more discoveries
	path, hasMore = exp.Next()
	if hasMore {
		fmt.Printf("Unexpected additional discovery: %s\n", path)
	}

	// Collect expanded paths
	expandedPaths, err := exp.Collect()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Expanded paths (%d):\n", len(expandedPaths))
	for i, p := range expandedPaths {
		fmt.Printf("  %d. %s\n", i+1, p)
	}
}

func multiLevelExample() {
	// Get expander from pool
	exp := expander.Get()
	defer expander.Release(exp)

	// Add multi-level wildcard path
	err := exp.Add([]string{
		"InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Original path: InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable\n")

	// Discovery iteration
	iteration := 1
	for {
		path, hasMore := exp.Next()
		if !hasMore {
			break
		}

		fmt.Printf("Iteration %d - Discovery path: %s\n", iteration, path)

		// Simulate discovery
		parameterNames := simulateDiscovery(path)
		fmt.Printf("  Discovered parameter names: %v\n", parameterNames)

		err := exp.Register(parameterNames)
		if err != nil {
			log.Printf("  Error registering results: %v", err)
			continue
		}

		iteration++
	}

	// Get final results
	expandedPaths, err := exp.Collect()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final expanded paths (%d):\n", len(expandedPaths))
	for i, p := range expandedPaths {
		fmt.Printf("  %d. %s\n", i+1, p)
	}
}

func commonAncestorExample() {
	exp := expander.Get()
	defer expander.Release(exp)

	// Add multiple paths with common ancestor
	err := exp.Add([]string{
		"Device.WiFi.AccessPoint.*.Enable",
		"Device.WiFi.AccessPoint.*.Status",
		"Device.WiFi.AccessPoint.*.SSID",
		"Device.WiFi.AccessPoint.*.Name",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Added 4 paths with common ancestor Device.WiFi.AccessPoint.*")

	// Should only need one discovery
	path, hasMore := exp.Next()
	fmt.Printf("Discovery path: %s\n", path)

	if hasMore {
		err := exp.Register([]string{
			"Device.WiFi.AccessPoint.1",
			"Device.WiFi.AccessPoint.2",
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Registered 2 access points")
	}

	// Check if more discoveries needed
	path, hasMore = exp.Next()
	if hasMore {
		fmt.Printf("Unexpected additional discovery: %s\n", path)
	} else {
		fmt.Println("No more discoveries needed - common ancestor optimization worked!")
	}

	expandedPaths, err := exp.Collect()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Expanded to %d paths (2 instances × 4 properties):\n", len(expandedPaths))
	for i, p := range expandedPaths {
		fmt.Printf("  %d. %s\n", i+1, p)
	}
}

func dynamicAdditionExample() {
	exp := expander.Get()
	defer expander.Release(exp)

	// Initial path
	err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added initial path: Device.WiFi.AccessPoint.*.Enable")

	// Discover and register
	_, _ = exp.Next()
	err = exp.Register([]string{
		"Device.WiFi.AccessPoint.1",
		"Device.WiFi.AccessPoint.2",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Check completion
	_, hasMore := exp.Next()
	if !hasMore {
		fmt.Println("Initial expansion complete")
	}

	// Dynamically add another path with same ancestor
	err = exp.Add([]string{"Device.WiFi.AccessPoint.*.Status"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Dynamically added: Device.WiFi.AccessPoint.*.Status")

	// Should not need new discovery (uses cache)
	_, hasMore = exp.Next()
	if hasMore {
		fmt.Println("ERROR: Should not need additional discovery!")
	} else {
		fmt.Println("No additional discovery needed - cache reused!")
	}

	expandedPaths, err := exp.Collect()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Final paths (%d):\n", len(expandedPaths))
	for i, p := range expandedPaths {
		fmt.Printf("  %d. %s\n", i+1, p)
	}
}

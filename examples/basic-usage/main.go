package main

import (
	"fmt"
	"log"

	expander "github.com/metalgrid/tr069-path-expander"
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
	fmt.Println("TR-069 Wildcard Expander - Basic Usage Example")
	fmt.Println("==============================================")

	// Example 1: Single wildcard expansion
	fmt.Println("\n1. Single Wildcard Example:")
	singleWildcardExample()

	// Example 2: Multi-level wildcard expansion
	fmt.Println("\n2. Multi-level Wildcard Example:")
	multiLevelExample()

	// Example 3: Error handling
	fmt.Println("\n3. Error Handling Example:")
	errorHandlingExample()
}

func singleWildcardExample() {
	// Create expander for single wildcard path
	exp, err := expander.New("Device.WiFi.AccessPoint.*.Enable")
	if err != nil {
		log.Fatal(err)
	}
	defer expander.Release(exp)

	fmt.Printf("Original path: Device.WiFi.AccessPoint.*.Enable\n")

	// Get discovery path
	path, hasMore := exp.NextDiscoveryPath()
	fmt.Printf("Discovery path: %s\n", path)

	if hasMore {
		// Simulate discovery - found 3 access points
		parameterNames := []string{
			"Device.WiFi.AccessPoint.1",
			"Device.WiFi.AccessPoint.2",
			"Device.WiFi.AccessPoint.3",
		}
		pathWithoutDot := path[:len(path)-1]

		err := exp.RegisterParameterNames(pathWithoutDot, parameterNames)
		if err != nil {
			log.Printf("Error registering indices: %v", err)
			return
		}

		fmt.Printf("Registered parameter names: %v\n", parameterNames)
	}

	// Check if expansion is complete
	if exp.IsComplete() {
		expandedPaths := exp.ExpandedPaths()
		fmt.Printf("Expanded paths (%d):\n", len(expandedPaths))
		for i, path := range expandedPaths {
			fmt.Printf("  %d. %s\n", i+1, path)
		}
	}
}

func multiLevelExample() {
	// Create expander for multi-level wildcard path
	exp, err := expander.New("InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable")
	if err != nil {
		log.Fatal(err)
	}
	defer expander.Release(exp)

	fmt.Printf("Original path: InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable\n")

	// Discovery iteration
	iteration := 1
	for !exp.IsComplete() {
		path, hasMore := exp.NextDiscoveryPath()
		if !hasMore {
			break
		}

		fmt.Printf("Iteration %d - Discovery path: %s\n", iteration, path)

		// Simulate discovery
		parameterNames := simulateDiscovery(path)
		pathWithoutDot := path[:len(path)-1]

		fmt.Printf("  Discovered parameter names: %v\n", parameterNames)

		err := exp.RegisterParameterNames(pathWithoutDot, parameterNames)
		if err != nil {
			log.Printf("  Error registering indices: %v", err)
			continue
		}

		iteration++
	}

	// Get final results
	if exp.IsComplete() {
		expandedPaths := exp.ExpandedPaths()
		fmt.Printf("Final expanded paths (%d):\n", len(expandedPaths))
		for i, path := range expandedPaths {
			fmt.Printf("  %d. %s\n", i+1, path)
		}
	}
}

func errorHandlingExample() {
	// Example 1: Invalid path
	fmt.Println("Testing invalid path...")
	exp, err := expander.New("")
	if err != nil {
		fmt.Printf("  Expected error for empty path: %v\n", err)
	} else {
		expander.Release(exp)
	}

	// Example 2: Path mismatch during registration
	fmt.Println("Testing path mismatch...")
	exp, err = expander.New("Device.Test.*.Value")
	if err != nil {
		log.Fatal(err)
	}
	defer expander.Release(exp)

	// Try to register parameter names for wrong path
	err = exp.RegisterParameterNames("Device.Wrong.Path", []string{"Device.Wrong.Path.1", "Device.Wrong.Path.2"})
	if err != nil {
		fmt.Printf("  Expected error for path mismatch: %v\n", err)
	}

	// Example 3: Registration after completion
	fmt.Println("Testing registration after completion...")

	// Complete the expansion properly first
	path, _ := exp.NextDiscoveryPath()
	pathWithoutDot := path[:len(path)-1]
	exp.RegisterParameterNames(pathWithoutDot, []string{pathWithoutDot + ".1"})

	// Now try to register again after completion
	err = exp.RegisterParameterNames(pathWithoutDot, []string{pathWithoutDot + ".2"})
	if err != nil {
		fmt.Printf("  Expected error for registration after completion: %v\n", err)
	}
}

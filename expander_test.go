package expander_test

import (
	"testing"

	expander "github.com/metalgrid/tr069-path-expander"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExpander(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Expander Suite")
}

var _ = Describe("TR-069 Path Expander", func() {
	var exp *expander.Expander

	AfterEach(func() {
		if exp != nil {
			expander.Release(exp)
			exp = nil
		}
	})

	Describe("Basic API", func() {
		Context("when getting an expander from the pool", func() {
			It("should return a fresh expander instance", func() {
				exp = expander.Get()
				Expect(exp).NotTo(BeNil())

				// Should start empty
				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(BeEmpty())
			})
		})

		Context("when adding paths without wildcards", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should return the paths unchanged", func() {
				err := exp.Add([]string{
					"InternetGatewayDevice.LANDevice.1.Enable",
					"InternetGatewayDevice.LANDevice.2.Enable",
				})
				Expect(err).NotTo(HaveOccurred())

				// No discovery needed
				path, hasMore := exp.Next()
				Expect(path).To(BeEmpty())
				Expect(hasMore).To(BeFalse())

				// Should return the original paths
				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(ConsistOf(
					"InternetGatewayDevice.LANDevice.1.Enable",
					"InternetGatewayDevice.LANDevice.2.Enable",
				))
			})
		})

		Context("when adding an invalid path", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should return an error for empty path", func() {
				err := exp.Add([]string{""})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expander.ErrInvalidPath))
			})
		})
	})

	Describe("Single Wildcard Expansion", func() {
		Context("when adding a path with a single wildcard", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should provide the correct discovery path", func() {
				err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
				Expect(err).NotTo(HaveOccurred())

				path, hasMore := exp.Next()
				Expect(hasMore).To(BeTrue())
				Expect(path).To(Equal("Device.WiFi.AccessPoint."))
			})

			It("should expand paths after registration", func() {
				err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
				Expect(err).NotTo(HaveOccurred())

				// Get discovery path
				_, hasMore := exp.Next()
				Expect(hasMore).To(BeTrue())

				// Register results
				err = exp.Register([]string{
					"Device.WiFi.AccessPoint.1",
					"Device.WiFi.AccessPoint.2",
					"Device.WiFi.AccessPoint.3",
				})
				Expect(err).NotTo(HaveOccurred())

				// Should be complete
				_, hasMore = exp.Next()
				Expect(hasMore).To(BeFalse())

				// Collect expanded paths
				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(ConsistOf(
					"Device.WiFi.AccessPoint.1.Enable",
					"Device.WiFi.AccessPoint.2.Enable",
					"Device.WiFi.AccessPoint.3.Enable",
				))
			})

			It("should handle empty discovery results", func() {
				err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
				Expect(err).NotTo(HaveOccurred())

				_, _ = exp.Next()

				// Register empty results
				err = exp.Register([]string{})
				Expect(err).NotTo(HaveOccurred())

				// Should be complete with no paths
				_, hasMore := exp.Next()
				Expect(hasMore).To(BeFalse())

				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(BeEmpty())
			})
		})
	})

	Describe("Multi-level Wildcard Expansion", func() {
		Context("when adding paths with multiple wildcards", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should handle nested wildcard expansion", func() {
				err := exp.Add([]string{
					"InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable",
				})
				Expect(err).NotTo(HaveOccurred())

				// First level discovery
				path, hasMore := exp.Next()
				Expect(hasMore).To(BeTrue())
				Expect(path).To(Equal("InternetGatewayDevice.LANDevice."))

				// Register first level
				err = exp.Register([]string{
					"InternetGatewayDevice.LANDevice.1",
					"InternetGatewayDevice.LANDevice.2",
				})
				Expect(err).NotTo(HaveOccurred())

				// Second level discovery for LANDevice.1
				path, hasMore = exp.Next()
				Expect(hasMore).To(BeTrue())
				// Debug output
				if path != "InternetGatewayDevice.LANDevice.1.WLANConfiguration." {
					GinkgoT().Logf("Expected: InternetGatewayDevice.LANDevice.1.WLANConfiguration.")
					GinkgoT().Logf("Got: %s", path)
				}
				Expect(path).To(Equal("InternetGatewayDevice.LANDevice.1.WLANConfiguration."))

				// Register second level for LANDevice.1
				err = exp.Register([]string{
					"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1",
					"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2",
				})
				Expect(err).NotTo(HaveOccurred())

				// Second level discovery for LANDevice.2
				path, hasMore = exp.Next()
				Expect(hasMore).To(BeTrue())
				Expect(path).To(Equal("InternetGatewayDevice.LANDevice.2.WLANConfiguration."))

				// Register second level for LANDevice.2 (empty)
				err = exp.Register([]string{})
				Expect(err).NotTo(HaveOccurred())

				// Should be complete
				_, hasMore = exp.Next()
				Expect(hasMore).To(BeFalse())

				// Collect expanded paths
				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(ConsistOf(
					"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable",
					"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Enable",
				))
			})
		})
	})

	Describe("Common Ancestor Optimization", func() {
		Context("when adding multiple paths with common ancestors", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should only request common ancestor once", func() {
				err := exp.Add([]string{
					"Device.WiFi.AccessPoint.*.Enable",
					"Device.WiFi.AccessPoint.*.Status",
					"Device.WiFi.AccessPoint.*.Name",
				})
				Expect(err).NotTo(HaveOccurred())

				// Should only get one discovery path for the common wildcard
				path, hasMore := exp.Next()
				Expect(hasMore).To(BeTrue())
				Expect(path).To(Equal("Device.WiFi.AccessPoint."))

				// Register results
				err = exp.Register([]string{
					"Device.WiFi.AccessPoint.1",
					"Device.WiFi.AccessPoint.2",
				})
				Expect(err).NotTo(HaveOccurred())

				// Should be complete
				_, hasMore = exp.Next()
				Expect(hasMore).To(BeFalse())

				// Should expand all three properties for each instance
				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(ConsistOf(
					"Device.WiFi.AccessPoint.1.Enable",
					"Device.WiFi.AccessPoint.1.Name",
					"Device.WiFi.AccessPoint.1.Status",
					"Device.WiFi.AccessPoint.2.Enable",
					"Device.WiFi.AccessPoint.2.Name",
					"Device.WiFi.AccessPoint.2.Status",
				))
			})
		})
	})

	Describe("Dynamic Path Addition", func() {
		Context("when adding paths after initial expansion", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should reuse cache for common ancestors", func() {
				// Initial expansion
				err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
				Expect(err).NotTo(HaveOccurred())

				_, hasMore := exp.Next()
				Expect(hasMore).To(BeTrue())

				err = exp.Register([]string{
					"Device.WiFi.AccessPoint.1",
					"Device.WiFi.AccessPoint.2",
				})
				Expect(err).NotTo(HaveOccurred())

				_, hasMore = exp.Next()
				Expect(hasMore).To(BeFalse())

				initialPaths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(initialPaths).To(HaveLen(2))

				// Add another path with same ancestor
				err = exp.Add([]string{"Device.WiFi.AccessPoint.*.Status"})
				Expect(err).NotTo(HaveOccurred())

				// Should not need discovery (uses cache)
				_, hasMore = exp.Next()
				Expect(hasMore).To(BeFalse())

				// Should now have both Enable and Status
				finalPaths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(finalPaths).To(ConsistOf(
					"Device.WiFi.AccessPoint.1.Enable",
					"Device.WiFi.AccessPoint.1.Status",
					"Device.WiFi.AccessPoint.2.Enable",
					"Device.WiFi.AccessPoint.2.Status",
				))
			})

			It("should handle new ancestors that need discovery", func() {
				// Initial expansion
				err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
				Expect(err).NotTo(HaveOccurred())

				path, _ := exp.Next()
				err = exp.Register([]string{
					"Device.WiFi.AccessPoint.1",
				})
				Expect(err).NotTo(HaveOccurred())

				_, hasMore := exp.Next()
				Expect(hasMore).To(BeFalse())

				// Add path with different ancestor
				err = exp.Add([]string{"Device.Ethernet.Interface.*.Status"})
				Expect(err).NotTo(HaveOccurred())

				// Should need new discovery
				path, hasMore = exp.Next()
				Expect(hasMore).To(BeTrue())
				Expect(path).To(Equal("Device.Ethernet.Interface."))

				err = exp.Register([]string{
					"Device.Ethernet.Interface.1",
					"Device.Ethernet.Interface.2",
				})
				Expect(err).NotTo(HaveOccurred())

				path, hasMore = exp.Next()
				Expect(hasMore).To(BeFalse())

				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(ConsistOf(
					"Device.WiFi.AccessPoint.1.Enable",
					"Device.Ethernet.Interface.1.Status",
					"Device.Ethernet.Interface.2.Status",
				))
			})
		})
	})

	Describe("Duplicate Handling", func() {
		Context("when adding duplicate paths", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should not produce duplicate expanded paths", func() {
				err := exp.Add([]string{
					"Device.WiFi.AccessPoint.*.Enable",
					"Device.WiFi.AccessPoint.*.Enable", // Duplicate
				})
				Expect(err).NotTo(HaveOccurred())

				_, _ = exp.Next()
				err = exp.Register([]string{
					"Device.WiFi.AccessPoint.1",
					"Device.WiFi.AccessPoint.2",
				})
				Expect(err).NotTo(HaveOccurred())

				_, hasMore := exp.Next()
				Expect(hasMore).To(BeFalse())

				paths, err := exp.Collect()
				Expect(err).NotTo(HaveOccurred())
				Expect(paths).To(ConsistOf(
					"Device.WiFi.AccessPoint.1.Enable",
					"Device.WiFi.AccessPoint.2.Enable",
				))
				// Should not have duplicates
				Expect(paths).To(HaveLen(2))
			})
		})
	})

	Describe("Error Handling", func() {
		Context("when calling Collect before completion", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should return error if expansion not complete", func() {
				err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
				Expect(err).NotTo(HaveOccurred())

				// Try to collect before completion
				paths, err := exp.Collect()
				Expect(err).To(HaveOccurred())
				Expect(paths).To(BeNil())
			})
		})

		Context("when registering after completion", func() {
			BeforeEach(func() {
				exp = expander.Get()
			})

			It("should return error", func() {
				// Add path without wildcards (immediately complete)
				err := exp.Add([]string{"Device.WiFi.AccessPoint.1.Enable"})
				Expect(err).NotTo(HaveOccurred())

				_, hasMore := exp.Next()
				Expect(hasMore).To(BeFalse())

				// Try to register after completion
				err = exp.Register([]string{"Device.WiFi.AccessPoint.2"})
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expander.ErrAlreadyComplete))
			})
		})
	})

	Describe("Pool Management", func() {
		It("should provide fresh state after release and get", func() {
			// First usage
			exp = expander.Get()
			err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
			Expect(err).NotTo(HaveOccurred())

			_, _ = exp.Next()
			err = exp.Register([]string{"Device.WiFi.AccessPoint.1"})
			Expect(err).NotTo(HaveOccurred())

			expander.Release(exp)

			// Second usage - should have fresh state
			exp = expander.Get()
			paths, err := exp.Collect()
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(BeEmpty())
		})

		It("should allow reuse without release to maintain cache", func() {
			exp = expander.Get()

			// First operation
			err := exp.Add([]string{"Device.WiFi.AccessPoint.*.Enable"})
			Expect(err).NotTo(HaveOccurred())

			_, _ = exp.Next()
			err = exp.Register([]string{
				"Device.WiFi.AccessPoint.1",
				"Device.WiFi.AccessPoint.2",
			})
			Expect(err).NotTo(HaveOccurred())

			_, hasMore := exp.Next()
			Expect(hasMore).To(BeFalse())

			// Reuse same instance without release
			err = exp.Add([]string{"Device.WiFi.AccessPoint.*.Status"})
			Expect(err).NotTo(HaveOccurred())

			// Should use cache - no discovery needed
			_, hasMore = exp.Next()
			Expect(hasMore).To(BeFalse())

			paths, err := exp.Collect()
			Expect(err).NotTo(HaveOccurred())
			Expect(paths).To(HaveLen(4)) // 2 instances × 2 properties
		})
	})

	Describe("Complex Real-World Scenario", func() {
		BeforeEach(func() {
			exp = expander.Get()
		})

		It("should handle TR-069 WAN connection parameters", func() {
			err := exp.Add([]string{
				"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.Enable",
				"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.ConnectionStatus",
				"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.ConnectionType",
				"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.Name",
				"InternetGatewayDevice.WANDevice.*.WANConnectionDevice.*.WANIPConnection.*.Uptime",
			})
			Expect(err).NotTo(HaveOccurred())

			// First level: WANDevice
			path, hasMore := exp.Next()
			Expect(hasMore).To(BeTrue())
			Expect(path).To(Equal("InternetGatewayDevice.WANDevice."))

			err = exp.Register([]string{
				"InternetGatewayDevice.WANDevice.1",
				"InternetGatewayDevice.WANDevice.2",
			})
			Expect(err).NotTo(HaveOccurred())

			// Second level: WANConnectionDevice for WANDevice.1
			path, hasMore = exp.Next()
			Expect(hasMore).To(BeTrue())
			Expect(path).To(Equal("InternetGatewayDevice.WANDevice.1.WANConnectionDevice."))

			err = exp.Register([]string{
				"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1",
			})
			Expect(err).NotTo(HaveOccurred())

			// Second level: WANConnectionDevice for WANDevice.2
			path, hasMore = exp.Next()
			Expect(hasMore).To(BeTrue())
			Expect(path).To(Equal("InternetGatewayDevice.WANDevice.2.WANConnectionDevice."))

			err = exp.Register([]string{
				"InternetGatewayDevice.WANDevice.2.WANConnectionDevice.1",
				"InternetGatewayDevice.WANDevice.2.WANConnectionDevice.2",
			})
			Expect(err).NotTo(HaveOccurred())

			// Third level: WANIPConnection for each WANConnectionDevice
			for i := 0; i < 3; i++ {
				path, hasMore = exp.Next()
				Expect(hasMore).To(BeTrue())

				if i == 0 {
					// WANDevice.1.WANConnectionDevice.1
					Expect(path).To(Equal("InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection."))
					err = exp.Register([]string{
						"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1",
					})
				} else if i == 1 {
					// WANDevice.2.WANConnectionDevice.1
					Expect(path).To(Equal("InternetGatewayDevice.WANDevice.2.WANConnectionDevice.1.WANIPConnection."))
					err = exp.Register([]string{
						"InternetGatewayDevice.WANDevice.2.WANConnectionDevice.1.WANIPConnection.1",
						"InternetGatewayDevice.WANDevice.2.WANConnectionDevice.1.WANIPConnection.2",
					})
				} else {
					// WANDevice.2.WANConnectionDevice.2
					Expect(path).To(Equal("InternetGatewayDevice.WANDevice.2.WANConnectionDevice.2.WANIPConnection."))
					err = exp.Register([]string{}) // No connections
				}
				Expect(err).NotTo(HaveOccurred())
			}

			// Should be complete
			path, hasMore = exp.Next()
			Expect(hasMore).To(BeFalse())

			// Collect all paths
			paths, err := exp.Collect()
			Expect(err).NotTo(HaveOccurred())

			// Should have 5 parameters for each connection:
			// WANDevice.1.WANConnectionDevice.1.WANIPConnection.1: 5 params
			// WANDevice.2.WANConnectionDevice.1.WANIPConnection.1: 5 params
			// WANDevice.2.WANConnectionDevice.1.WANIPConnection.2: 5 params
			// Total: 15 paths
			Expect(paths).To(HaveLen(15))

			// Verify some specific paths
			Expect(paths).To(ContainElements(
				"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.Enable",
				"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ConnectionStatus",
				"InternetGatewayDevice.WANDevice.2.WANConnectionDevice.1.WANIPConnection.2.Uptime",
			))
		})
	})
})

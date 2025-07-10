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

var _ = Describe("WildcardExpander", func() {
	var exp expander.Expander
	var err error

	AfterEach(func() {
		if exp != nil {
			expander.Release(exp)
		}
	})

	Context("when initialized with a path without wildcards", func() {
		BeforeEach(func() {
			exp, err = expander.New("InternetGatewayDevice.LANDevice.1.Enable")
		})

		It("should initialize successfully", func() {
			Expect(err).NotTo(HaveOccurred())
			Expect(exp).NotTo(BeNil())
		})

		It("should be complete immediately", func() {
			Expect(exp.IsComplete()).To(BeTrue())
		})

		It("should return no discovery paths", func() {
			path, hasMore := exp.NextDiscoveryPath()
			Expect(path).To(BeEmpty())
			Expect(hasMore).To(BeFalse())
		})

		It("should return the original path as expanded", func() {
			paths := exp.ExpandedPaths()
			Expect(paths).To(HaveLen(1))
			Expect(paths[0]).To(Equal("InternetGatewayDevice.LANDevice.1.Enable"))
		})
	})

	Context("when initialized with a single wildcard path", func() {
		BeforeEach(func() {
			exp, err = expander.New("InternetGatewayDevice.LANDevice.*.Enable")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when parameter names are registered", func() {
			BeforeEach(func() {
				discoveryPath, _ := exp.NextDiscoveryPath()
				Expect(discoveryPath).To(Equal("InternetGatewayDevice.LANDevice."))
				parameterNames := []string{
					"InternetGatewayDevice.LANDevice.1",
					"InternetGatewayDevice.LANDevice.2",
					"InternetGatewayDevice.LANDevice.3",
				}
				err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice", parameterNames)
			})

			It("should register parameter names successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should be complete after registration", func() {
				Expect(exp.IsComplete()).To(BeTrue())
			})

			It("should return no more discovery paths", func() {
				path, hasMore := exp.NextDiscoveryPath()
				Expect(path).To(BeEmpty())
				Expect(hasMore).To(BeFalse())
			})

			It("should return expanded paths", func() {
				paths := exp.ExpandedPaths()
				Expect(paths).To(HaveLen(3))
				Expect(paths).To(ContainElements(
					"InternetGatewayDevice.LANDevice.1.Enable",
					"InternetGatewayDevice.LANDevice.2.Enable",
					"InternetGatewayDevice.LANDevice.3.Enable",
				))
			})
		})

		Context("when empty parameter names are registered", func() {
			BeforeEach(func() {
				discoveryPath, _ := exp.NextDiscoveryPath()
				// Empty parameter names array (no instances found)
				err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice", []string{})
				_ = discoveryPath // Use the variable to avoid unused error
			})

			It("should register empty parameter names successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should be complete after registration", func() {
				Expect(exp.IsComplete()).To(BeTrue())
			})

			It("should return no expanded paths", func() {
				paths := exp.ExpandedPaths()
				Expect(paths).To(BeEmpty())
			})
		})
	})

	Context("when initialized with multiple wildcard path", func() {
		BeforeEach(func() {
			exp, err = expander.New("InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when first level parameter names are registered", func() {
			BeforeEach(func() {
				discoveryPath, _ := exp.NextDiscoveryPath()
				Expect(discoveryPath).To(Equal("InternetGatewayDevice.LANDevice."))
				parameterNames := []string{
					"InternetGatewayDevice.LANDevice.1",
					"InternetGatewayDevice.LANDevice.2",
					"InternetGatewayDevice.LANDevice.3",
				}
				err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice", parameterNames)
			})

			It("should register parameter names successfully", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not be complete yet", func() {
				Expect(exp.IsComplete()).To(BeFalse())
			})

			It("should return second level discovery paths", func() {
				var secondLevelPaths []string
				for {
					path, hasMore := exp.NextDiscoveryPath()
					if !hasMore {
						break
					}
					secondLevelPaths = append(secondLevelPaths, path)
				}

				Expect(secondLevelPaths).To(HaveLen(3))
				Expect(secondLevelPaths).To(ContainElements(
					"InternetGatewayDevice.LANDevice.1.WLANConfiguration.",
					"InternetGatewayDevice.LANDevice.2.WLANConfiguration.",
					"InternetGatewayDevice.LANDevice.3.WLANConfiguration.",
				))
			})

			Context("when second level parameter names are registered", func() {
				BeforeEach(func() {
					// First, consume all discovery paths
					var secondLevelPaths []string
					for {
						path, hasMore := exp.NextDiscoveryPath()
						if !hasMore {
							break
						}
						secondLevelPaths = append(secondLevelPaths, path)
					}

					// Register second level - some with parameters, some empty
					err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.1.WLANConfiguration", []string{
						"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1",
						"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2",
					})
					Expect(err).NotTo(HaveOccurred())

					err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.2.WLANConfiguration", []string{})
					Expect(err).NotTo(HaveOccurred())

					err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.3.WLANConfiguration", []string{})
					Expect(err).NotTo(HaveOccurred())
				})

				It("should be complete after all registrations", func() {
					Expect(exp.IsComplete()).To(BeTrue())
				})

				It("should return expanded paths only for paths with parameter names", func() {
					paths := exp.ExpandedPaths()
					Expect(paths).To(HaveLen(2))
					Expect(paths).To(ContainElements(
						"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable",
						"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Enable",
					))
				})
			})
		})
	})

	Context("when handling edge cases", func() {
		Context("with invalid paths", func() {
			It("should return error for empty path", func() {
				exp, err := expander.New("")
				Expect(err).To(HaveOccurred())
				Expect(exp).To(BeNil())
			})
		})

		Context("with registration after completion", func() {
			BeforeEach(func() {
				exp, err = expander.New("InternetGatewayDevice.LANDevice.*.Enable")
				Expect(err).NotTo(HaveOccurred())

				// Complete the expansion
				discoveryPath, _ := exp.NextDiscoveryPath()
				pathWithoutDot := discoveryPath[:len(discoveryPath)-1]
				parameterNames := []string{pathWithoutDot + ".1"}
				err = exp.RegisterParameterNames(pathWithoutDot, parameterNames)
				Expect(err).NotTo(HaveOccurred())
				Expect(exp.IsComplete()).To(BeTrue())
			})

			It("should return error when trying to register after completion", func() {
				err := exp.RegisterParameterNames("InternetGatewayDevice.LANDevice", []string{
					"InternetGatewayDevice.LANDevice.2",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("expansion is already complete"))
			})
		})

		Context("with path mismatch", func() {
			BeforeEach(func() {
				exp, err = expander.New("InternetGatewayDevice.LANDevice.*.Enable")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return error for mismatched path", func() {
				err := exp.RegisterParameterNames("InternetGatewayDevice.WrongPath", []string{
					"InternetGatewayDevice.WrongPath.1",
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("path mismatch"))
			})
		})
	})

	Context("when testing the example from requirements", func() {
		BeforeEach(func() {
			exp, err = expander.New("InternetGatewayDevice.LANDevice.*.WLANConfiguration.*.Enable")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should follow the exact workflow from the requirements", func() {
			// Step 2: Retrieve next path for discovery
			discoveryPath, hasMore := exp.NextDiscoveryPath()
			Expect(discoveryPath).To(Equal("InternetGatewayDevice.LANDevice."))
			Expect(hasMore).To(BeTrue())

			// Step 3: Register response
			err := exp.RegisterParameterNames("InternetGatewayDevice.LANDevice", []string{
				"InternetGatewayDevice.LANDevice.1",
				"InternetGatewayDevice.LANDevice.2",
				"InternetGatewayDevice.LANDevice.3",
			})
			Expect(err).NotTo(HaveOccurred())

			// Step 4: Retrieve next paths for discovery
			var secondLevelPaths []string
			for {
				path, hasMore := exp.NextDiscoveryPath()
				if !hasMore {
					break
				}
				secondLevelPaths = append(secondLevelPaths, path)
			}

			Expect(secondLevelPaths).To(HaveLen(3))
			Expect(secondLevelPaths).To(ContainElements(
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.",
				"InternetGatewayDevice.LANDevice.2.WLANConfiguration.",
				"InternetGatewayDevice.LANDevice.3.WLANConfiguration.",
			))

			// Step 5: Register responses for each value
			err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.1.WLANConfiguration", []string{
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1",
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2",
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.3",
			})
			Expect(err).NotTo(HaveOccurred())

			err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.2.WLANConfiguration", []string{})
			Expect(err).NotTo(HaveOccurred())

			err = exp.RegisterParameterNames("InternetGatewayDevice.LANDevice.3.WLANConfiguration", []string{})
			Expect(err).NotTo(HaveOccurred())

			// Step 6: Retrieve next path for discovery (should be empty)
			finalPath, hasMore := exp.NextDiscoveryPath()
			Expect(finalPath).To(BeEmpty())
			Expect(hasMore).To(BeFalse())
			Expect(exp.IsComplete()).To(BeTrue())

			// Step 7: Retrieve parameter names for value retrieval
			expandedPaths := exp.ExpandedPaths()
			Expect(expandedPaths).To(HaveLen(3))
			Expect(expandedPaths).To(ContainElements(
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.Enable",
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.2.Enable",
				"InternetGatewayDevice.LANDevice.1.WLANConfiguration.3.Enable",
			))
		})
	})

	Context("when testing parameter name extraction", func() {
		BeforeEach(func() {
			exp, err = expander.New("Device.WiFi.AccessPoint.*.Enable")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should extract indices from parameter names with sub-parameters", func() {
			discoveryPath, _ := exp.NextDiscoveryPath()
			pathWithoutDot := discoveryPath[:len(discoveryPath)-1]

			// Realistic TR-069 response with sub-parameters
			parameterNames := []string{
				"Device.WiFi.AccessPoint.1.Enable",
				"Device.WiFi.AccessPoint.1.SSID",
				"Device.WiFi.AccessPoint.1.Security.Mode",
				"Device.WiFi.AccessPoint.2.Enable",
				"Device.WiFi.AccessPoint.2.SSID",
				"Device.WiFi.AccessPoint.3.Enable",
			}

			err := exp.RegisterParameterNames(pathWithoutDot, parameterNames)
			Expect(err).NotTo(HaveOccurred())

			expandedPaths := exp.ExpandedPaths()
			Expect(expandedPaths).To(HaveLen(3))
			Expect(expandedPaths).To(ContainElements(
				"Device.WiFi.AccessPoint.1.Enable",
				"Device.WiFi.AccessPoint.2.Enable",
				"Device.WiFi.AccessPoint.3.Enable",
			))
		})

		It("should handle non-numeric indices gracefully", func() {
			discoveryPath, _ := exp.NextDiscoveryPath()
			pathWithoutDot := discoveryPath[:len(discoveryPath)-1]

			// Parameter names with non-numeric parts (should be ignored)
			parameterNames := []string{
				"Device.WiFi.AccessPoint.1.Enable",
				"Device.WiFi.AccessPoint.2.Enable",
				"Device.WiFi.AccessPoint.Status",   // Non-numeric
				"Device.WiFi.AccessPoint.MaxCount", // Non-numeric
			}

			err := exp.RegisterParameterNames(pathWithoutDot, parameterNames)
			Expect(err).NotTo(HaveOccurred())

			expandedPaths := exp.ExpandedPaths()
			Expect(expandedPaths).To(HaveLen(2))
			Expect(expandedPaths).To(ContainElements(
				"Device.WiFi.AccessPoint.1.Enable",
				"Device.WiFi.AccessPoint.2.Enable",
			))
		})
	})
})

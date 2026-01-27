package service_test

import (
	"github.com/dcm-project/policy-manager/internal/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filter", func() {
	Describe("ParseFilter", func() {
		Context("with valid single condition filters", func() {
			It("should parse policy_type='GLOBAL'", func() {
				filter, err := service.ParseFilter("policy_type='GLOBAL'")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.PolicyType).NotTo(BeNil())
				Expect(*filter.PolicyType).To(Equal("GLOBAL"))
				Expect(filter.Enabled).To(BeNil())
			})

			It("should parse policy_type='USER'", func() {
				filter, err := service.ParseFilter("policy_type='USER'")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.PolicyType).NotTo(BeNil())
				Expect(*filter.PolicyType).To(Equal("USER"))
				Expect(filter.Enabled).To(BeNil())
			})

			It("should parse enabled=true", func() {
				filter, err := service.ParseFilter("enabled=true")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.Enabled).NotTo(BeNil())
				Expect(*filter.Enabled).To(BeTrue())
				Expect(filter.PolicyType).To(BeNil())
			})

			It("should parse enabled=false", func() {
				filter, err := service.ParseFilter("enabled=false")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.Enabled).NotTo(BeNil())
				Expect(*filter.Enabled).To(BeFalse())
				Expect(filter.PolicyType).To(BeNil())
			})

			It("should handle whitespace around operators", func() {
				filter, err := service.ParseFilter("policy_type = 'GLOBAL'")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.PolicyType).NotTo(BeNil())
				Expect(*filter.PolicyType).To(Equal("GLOBAL"))
			})

			It("should handle whitespace around boolean values", func() {
				filter, err := service.ParseFilter("enabled = true")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.Enabled).NotTo(BeNil())
				Expect(*filter.Enabled).To(BeTrue())
			})
		})

		Context("with valid combined filters", func() {
			It("should parse policy_type='GLOBAL' AND enabled=true", func() {
				filter, err := service.ParseFilter("policy_type='GLOBAL' AND enabled=true")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.PolicyType).NotTo(BeNil())
				Expect(*filter.PolicyType).To(Equal("GLOBAL"))
				Expect(filter.Enabled).NotTo(BeNil())
				Expect(*filter.Enabled).To(BeTrue())
			})

			It("should parse enabled=false AND policy_type='USER'", func() {
				filter, err := service.ParseFilter("enabled=false AND policy_type='USER'")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.PolicyType).NotTo(BeNil())
				Expect(*filter.PolicyType).To(Equal("USER"))
				Expect(filter.Enabled).NotTo(BeNil())
				Expect(*filter.Enabled).To(BeFalse())
			})

			It("should parse policy_type='USER' AND enabled=true", func() {
				filter, err := service.ParseFilter("policy_type='USER' AND enabled=true")

				Expect(err).To(BeNil())
				Expect(filter).NotTo(BeNil())
				Expect(filter.PolicyType).NotTo(BeNil())
				Expect(*filter.PolicyType).To(Equal("USER"))
				Expect(filter.Enabled).NotTo(BeNil())
				Expect(*filter.Enabled).To(BeTrue())
			})
		})

		Context("with invalid filter expressions", func() {
			It("should return error for empty filter", func() {
				filter, err := service.ParseFilter("")

				Expect(err).To(BeNil())
				Expect(filter).To(BeNil())
			})

			It("should return error for unsupported field", func() {
				_, err := service.ParseFilter("display_name='test'")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid filter expression"))
			})

			It("should return error for invalid policy_type value", func() {
				_, err := service.ParseFilter("policy_type='INVALID'")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid filter expression"))
			})

			It("should return error for invalid enabled value", func() {
				_, err := service.ParseFilter("enabled=maybe")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid filter expression"))
			})

			It("should return error for random text", func() {
				_, err := service.ParseFilter("this is not a valid filter")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid filter expression"))
			})

			It("should return error for multiple AND operators", func() {
				_, err := service.ParseFilter("policy_type='GLOBAL' AND enabled=true AND priority=1")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid filter expression"))
			})
		})
	})
})

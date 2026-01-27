package service_test

import (
	"github.com/dcm-project/policy-manager/internal/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("OrderBy", func() {
	Describe("ParseOrderBy", func() {
		Context("with valid single field orders", func() {
			It("should parse priority asc", func() {
				result, err := service.ParseOrderBy("priority asc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority ASC"))
			})

			It("should parse priority desc", func() {
				result, err := service.ParseOrderBy("priority desc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority DESC"))
			})

			It("should parse display_name asc", func() {
				result, err := service.ParseOrderBy("display_name asc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("display_name ASC"))
			})

			It("should parse display_name desc", func() {
				result, err := service.ParseOrderBy("display_name desc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("display_name DESC"))
			})

			It("should parse create_time asc", func() {
				result, err := service.ParseOrderBy("create_time asc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("create_time ASC"))
			})

			It("should parse create_time desc", func() {
				result, err := service.ParseOrderBy("create_time desc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("create_time DESC"))
			})

			It("should default to ASC when direction not specified", func() {
				result, err := service.ParseOrderBy("priority")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority ASC"))
			})

			It("should handle case-insensitive direction", func() {
				result, err := service.ParseOrderBy("priority DESC")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority DESC"))
			})

			It("should handle mixed case direction", func() {
				result, err := service.ParseOrderBy("priority DeSc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority DESC"))
			})
		})

		Context("with valid multiple field orders", func() {
			It("should parse create_time desc,priority asc", func() {
				result, err := service.ParseOrderBy("create_time desc,priority asc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("create_time DESC, priority ASC"))
			})

			It("should parse priority asc,display_name desc", func() {
				result, err := service.ParseOrderBy("priority asc,display_name desc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority ASC, display_name DESC"))
			})

			It("should parse three fields", func() {
				result, err := service.ParseOrderBy("priority asc,create_time desc,display_name asc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority ASC, create_time DESC, display_name ASC"))
			})

			It("should handle whitespace around commas", func() {
				result, err := service.ParseOrderBy("priority asc , display_name desc")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority ASC, display_name DESC"))
			})

			It("should handle multiple fields without direction", func() {
				result, err := service.ParseOrderBy("priority,display_name")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("priority ASC, display_name ASC"))
			})
		})

		Context("with empty or default input", func() {
			It("should return default for empty string", func() {
				result, err := service.ParseOrderBy("")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("policy_type ASC, priority ASC, id ASC"))
			})

			It("should return default for whitespace only", func() {
				result, err := service.ParseOrderBy("   ")

				Expect(err).To(BeNil())
				Expect(result).To(Equal("policy_type ASC, priority ASC, id ASC"))
			})
		})

		Context("with invalid input", func() {
			It("should return error for unsupported field", func() {
				_, err := service.ParseOrderBy("invalid_field asc")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid order_by field"))
			})

			It("should return error for invalid direction", func() {
				_, err := service.ParseOrderBy("priority up")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid order_by direction"))
			})

			It("should return error for too many tokens", func() {
				_, err := service.ParseOrderBy("priority asc extra")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid order_by format"))
			})

			It("should return error for unsupported field in multi-field order", func() {
				_, err := service.ParseOrderBy("priority asc,invalid desc")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid order_by field"))
			})

			It("should return error for invalid direction in multi-field order", func() {
				_, err := service.ParseOrderBy("priority asc,display_name wrong")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("Invalid order_by direction"))
			})
		})
	})
})

package service_test

import (
	"time"

	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/service"
	"github.com/dcm-project/policy-manager/internal/store/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Converter", func() {
	Describe("APIToDBModel", func() {
		It("should convert API model to DB model with all fields", func() {
			description := "Test description"
			enabled := true
			priority := int32(100)
			labelSelector := map[string]string{"env": "prod"}

			apiPolicy := v1alpha1.Policy{
				DisplayName:   "Test Policy",
				PolicyType:    v1alpha1.GLOBAL,
				RegoCode:      "package test\ndefault allow = true",
				Description:   &description,
				Enabled:       &enabled,
				Priority:      &priority,
				LabelSelector: &labelSelector,
			}

			dbPolicy := service.APIToDBModel(apiPolicy, "test-id")

			Expect(dbPolicy.ID).To(Equal("test-id"))
			Expect(dbPolicy.DisplayName).To(Equal("Test Policy"))
			Expect(dbPolicy.PolicyType).To(Equal("GLOBAL"))
			Expect(dbPolicy.Description).To(Equal("Test description"))
			Expect(dbPolicy.Enabled).To(BeTrue())
			Expect(dbPolicy.Priority).To(Equal(int32(100)))
			Expect(dbPolicy.LabelSelector).To(Equal(map[string]string{"env": "prod"}))
		})

		It("should use default values for optional fields", func() {
			apiPolicy := v1alpha1.Policy{
				DisplayName: "Test Policy",
				PolicyType:  v1alpha1.USER,
				RegoCode:    "package test",
			}

			dbPolicy := service.APIToDBModel(apiPolicy, "test-id")

			Expect(dbPolicy.ID).To(Equal("test-id"))
			Expect(dbPolicy.DisplayName).To(Equal("Test Policy"))
			Expect(dbPolicy.PolicyType).To(Equal("USER"))
			Expect(dbPolicy.Description).To(BeEmpty())
			Expect(dbPolicy.Enabled).To(BeTrue())       // Default value
			Expect(dbPolicy.Priority).To(Equal(int32(500))) // Default value
			Expect(dbPolicy.LabelSelector).To(BeNil())
		})

		It("should strip RegoCode from API model", func() {
			apiPolicy := v1alpha1.Policy{
				DisplayName: "Test Policy",
				PolicyType:  v1alpha1.GLOBAL,
				RegoCode:    "package test\ndefault allow = true",
			}

			dbPolicy := service.APIToDBModel(apiPolicy, "test-id")

			// Verify RegoCode is not in the DB model (no field exists)
			Expect(dbPolicy.ID).NotTo(BeEmpty())
			Expect(dbPolicy.DisplayName).NotTo(BeEmpty())
		})

		It("should handle empty label selector", func() {
			apiPolicy := v1alpha1.Policy{
				DisplayName:   "Test Policy",
				PolicyType:    v1alpha1.GLOBAL,
				RegoCode:      "package test",
				LabelSelector: &map[string]string{},
			}

			dbPolicy := service.APIToDBModel(apiPolicy, "test-id")

			Expect(dbPolicy.LabelSelector).To(Equal(map[string]string{}))
		})

		It("should handle explicit false for enabled", func() {
			enabled := false
			apiPolicy := v1alpha1.Policy{
				DisplayName: "Test Policy",
				PolicyType:  v1alpha1.GLOBAL,
				RegoCode:    "package test",
				Enabled:     &enabled,
			}

			dbPolicy := service.APIToDBModel(apiPolicy, "test-id")

			Expect(dbPolicy.Enabled).To(BeFalse())
		})

		It("should handle priority value of 1", func() {
			priority := int32(1)
			apiPolicy := v1alpha1.Policy{
				DisplayName: "Test Policy",
				PolicyType:  v1alpha1.GLOBAL,
				RegoCode:    "package test",
				Priority:    &priority,
			}

			dbPolicy := service.APIToDBModel(apiPolicy, "test-id")

			Expect(dbPolicy.Priority).To(Equal(int32(1)))
		})
	})

	Describe("DBToAPIModel", func() {
		It("should convert DB model to API model with all fields", func() {
			now := time.Now()
			dbPolicy := &model.Policy{
				ID:            "test-id",
				DisplayName:   "Test Policy",
				Description:   "Test description",
				PolicyType:    "GLOBAL",
				LabelSelector: map[string]string{"env": "prod"},
				Priority:      100,
				Enabled:       true,
				CreateTime:    now,
				UpdateTime:    now,
			}

			apiPolicy := service.DBToAPIModel(dbPolicy)

			Expect(*apiPolicy.Id).To(Equal("test-id"))
			Expect(*apiPolicy.Path).To(Equal("policies/test-id"))
			Expect(apiPolicy.DisplayName).To(Equal("Test Policy"))
			Expect(*apiPolicy.Description).To(Equal("Test description"))
			Expect(apiPolicy.PolicyType).To(Equal(v1alpha1.GLOBAL))
			Expect(*apiPolicy.LabelSelector).To(Equal(map[string]string{"env": "prod"}))
			Expect(*apiPolicy.Priority).To(Equal(int32(100)))
			Expect(*apiPolicy.Enabled).To(BeTrue())
			Expect(*apiPolicy.CreateTime).To(Equal(now))
			Expect(*apiPolicy.UpdateTime).To(Equal(now))
			Expect(apiPolicy.RegoCode).To(Equal("")) // Should be empty
		})

		It("should set path field correctly", func() {
			dbPolicy := &model.Policy{
				ID:          "my-policy-123",
				DisplayName: "Test",
				PolicyType:  "GLOBAL",
				Priority:    500,
				Enabled:     true,
				CreateTime:  time.Now(),
				UpdateTime:  time.Now(),
			}

			apiPolicy := service.DBToAPIModel(dbPolicy)

			Expect(*apiPolicy.Path).To(Equal("policies/my-policy-123"))
		})

		It("should return empty RegoCode", func() {
			dbPolicy := &model.Policy{
				ID:          "test-id",
				DisplayName: "Test",
				PolicyType:  "GLOBAL",
				Priority:    500,
				Enabled:     true,
				CreateTime:  time.Now(),
				UpdateTime:  time.Now(),
			}

			apiPolicy := service.DBToAPIModel(dbPolicy)

			Expect(apiPolicy.RegoCode).To(Equal(""))
		})

		It("should omit description if empty", func() {
			dbPolicy := &model.Policy{
				ID:          "test-id",
				DisplayName: "Test",
				Description: "",
				PolicyType:  "GLOBAL",
				Priority:    500,
				Enabled:     true,
				CreateTime:  time.Now(),
				UpdateTime:  time.Now(),
			}

			apiPolicy := service.DBToAPIModel(dbPolicy)

			Expect(apiPolicy.Description).To(BeNil())
		})

		It("should omit label selector if empty", func() {
			dbPolicy := &model.Policy{
				ID:            "test-id",
				DisplayName:   "Test",
				PolicyType:    "GLOBAL",
				LabelSelector: map[string]string{},
				Priority:      500,
				Enabled:       true,
				CreateTime:    time.Now(),
				UpdateTime:    time.Now(),
			}

			apiPolicy := service.DBToAPIModel(dbPolicy)

			Expect(apiPolicy.LabelSelector).To(BeNil())
		})

		It("should convert USER policy type correctly", func() {
			dbPolicy := &model.Policy{
				ID:          "test-id",
				DisplayName: "Test",
				PolicyType:  "USER",
				Priority:    500,
				Enabled:     true,
				CreateTime:  time.Now(),
				UpdateTime:  time.Now(),
			}

			apiPolicy := service.DBToAPIModel(dbPolicy)

			Expect(apiPolicy.PolicyType).To(Equal(v1alpha1.USER))
		})
	})
})

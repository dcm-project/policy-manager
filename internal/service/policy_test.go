package service_test

import (
	"context"

	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/service"
	"github.com/dcm-project/policy-manager/internal/store"
	"github.com/dcm-project/policy-manager/internal/store/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func strPtr(s string) *string { return &s }

func policyTypePtr(t v1alpha1.PolicyPolicyType) *v1alpha1.PolicyPolicyType { return &t }

var _ = Describe("PolicyService", func() {
	var (
		db            *gorm.DB
		dataStore     store.Store
		policyService service.PolicyService
		ctx           context.Context
	)

	BeforeEach(func() {
		var err error
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(db.AutoMigrate(&model.Policy{})).To(Succeed())

		dataStore = store.NewStore(db)
		policyService = service.NewPolicyService(dataStore)
		ctx = context.Background()
	})

	AfterEach(func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	})

	Describe("CreatePolicy", func() {
		It("should create policy with client-specified ID", func() {
			clientID := "my-custom-policy"
			regoCode := "package test\ndefault allow = true"

			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    &regoCode,
			}

			created, err := policyService.CreatePolicy(ctx, policy, &clientID)

			Expect(err).To(BeNil())
			Expect(created).NotTo(BeNil())
			Expect(*created.Id).To(Equal("my-custom-policy"))
			Expect(*created.Path).To(Equal("policies/my-custom-policy"))
			Expect(created.DisplayName).NotTo(BeNil())
			Expect(*created.DisplayName).To(Equal("Test Policy"))
			Expect(created.PolicyType).NotTo(BeNil())
			Expect(*created.PolicyType).To(Equal(v1alpha1.GLOBAL))
			Expect(created.RegoCode).NotTo(BeNil())
			Expect(*created.RegoCode).To(Equal(""))         // Should be empty
			Expect(*created.Enabled).To(BeTrue())           // Default value
			Expect(*created.Priority).To(Equal(int32(500))) // Default value
		})

		It("should create policy with server-generated UUID", func() {
			regoCode := "package test\ndefault allow = true"

			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.USER),
				RegoCode:    &regoCode,
			}

			created, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).To(BeNil())
			Expect(created).NotTo(BeNil())
			Expect(*created.Id).NotTo(BeEmpty())
			Expect(*created.Path).To(HavePrefix("policies/"))
			// UUID format validation
			Expect(*created.Id).To(MatchRegexp(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`))
		})

		It("should validate RegoCode is non-empty", func() {
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr(""),
			}

			_, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("rego_code is required"))
		})

		It("should validate priority is at least 1", func() {
			priority := int32(0)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}

			_, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("priority must be between 1 and 1000"))
		})

		It("should reject negative priority", func() {
			priority := int32(-1)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}

			_, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("priority must be between 1 and 1000"))
		})

		It("should accept priority at minimum (1)", func() {
			priority := int32(1)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy Min Priority"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}

			created, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).To(BeNil())
			Expect(created).NotTo(BeNil())
			Expect(*created.Priority).To(Equal(int32(1)))
		})

		It("should accept priority at maximum (1000)", func() {
			priority := int32(1000)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy Max Priority"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}

			created, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).To(BeNil())
			Expect(created).NotTo(BeNil())
			Expect(*created.Priority).To(Equal(int32(1000)))
		})

		It("should reject priority above maximum (1001)", func() {
			priority := int32(1001)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}

			_, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("priority must be between 1 and 1000"))
		})

		It("should accept priority in mid-range (500)", func() {
			priority := int32(500)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy Mid Priority"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}

			created, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).To(BeNil())
			Expect(created).NotTo(BeNil())
			Expect(*created.Priority).To(Equal(int32(500)))
		})

		It("should validate RegoCode is not just whitespace", func() {
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("   \n\t  "),
			}

			_, err := policyService.CreatePolicy(ctx, policy, nil)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
		})

		It("should validate ID format per AEP-122", func() {
			invalidID := "Invalid-ID-With-CAPS"
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
			}

			_, err := policyService.CreatePolicy(ctx, policy, &invalidID)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("Invalid policy ID format"))
		})

		It("should return AlreadyExists error for duplicate ID", func() {
			clientID := "duplicate-policy"
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
			}

			// Create first policy
			_, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			// Try to create duplicate
			_, err = policyService.CreatePolicy(ctx, policy, &clientID)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeAlreadyExists))
		})

		It("should return AlreadyExists when creating two policies with same display_name and policy_type", func() {
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Unique Display Name"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
			}
			id1 := "policy-dn-1"
			_, err := policyService.CreatePolicy(ctx, policy, &id1)
			Expect(err).To(BeNil())

			id2 := "policy-dn-2"
			_, err = policyService.CreatePolicy(ctx, policy, &id2)
			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeAlreadyExists))
			Expect(serviceErr.Message).To(ContainSubstring("Policy display name and policy type"))
		})

		It("should return AlreadyExists when creating two policies with same priority and policy_type", func() {
			priority := int32(100)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Policy One"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}
			id1 := "policy-prio-1"
			_, err := policyService.CreatePolicy(ctx, policy, &id1)
			Expect(err).To(BeNil())

			policy2 := v1alpha1.Policy{
				DisplayName: strPtr("Policy Two"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}
			id2 := "policy-prio-2"
			_, err = policyService.CreatePolicy(ctx, policy2, &id2)
			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeAlreadyExists))
			Expect(serviceErr.Message).To(ContainSubstring("Policy priority and policy type"))
		})

		It("should use default values for optional fields", func() {
			clientID := "defaults-test"
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
			}

			created, err := policyService.CreatePolicy(ctx, policy, &clientID)

			Expect(err).To(BeNil())
			Expect(*created.Enabled).To(BeTrue())
			Expect(*created.Priority).To(Equal(int32(500)))
		})

		It("should honor explicit values for optional fields", func() {
			clientID := "explicit-values"
			enabled := false
			priority := int32(100)
			description := "Custom description"
			labelSelector := map[string]string{"env": "prod"}

			policy := v1alpha1.Policy{
				DisplayName:   strPtr("Test Policy"),
				PolicyType:    policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:      strPtr("package test"),
				Enabled:       &enabled,
				Priority:      &priority,
				Description:   &description,
				LabelSelector: &labelSelector,
			}

			created, err := policyService.CreatePolicy(ctx, policy, &clientID)

			Expect(err).To(BeNil())
			Expect(*created.Enabled).To(BeFalse())
			Expect(*created.Priority).To(Equal(int32(100)))
			Expect(*created.Description).To(Equal("Custom description"))
			Expect(*created.LabelSelector).To(Equal(map[string]string{"env": "prod"}))
		})
	})

	Describe("GetPolicy", func() {
		It("should get existing policy", func() {
			// Create a policy first
			clientID := "get-test"
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
			}
			created, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			// Get the policy
			retrieved, err := policyService.GetPolicy(ctx, "get-test")

			Expect(err).To(BeNil())
			Expect(retrieved).NotTo(BeNil())
			Expect(*retrieved.Id).To(Equal("get-test"))
			Expect(*retrieved.Path).To(Equal("policies/get-test"))
			Expect(retrieved.DisplayName).NotTo(BeNil())
			Expect(*retrieved.DisplayName).To(Equal("Test Policy"))
			Expect(retrieved.RegoCode).NotTo(BeNil())
			Expect(*retrieved.RegoCode).To(Equal("")) // Should be empty
			Expect(retrieved.CreateTime).To(Equal(created.CreateTime))
		})

		It("should return NotFound error for non-existent policy", func() {
			_, err := policyService.GetPolicy(ctx, "non-existent")

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeNotFound))
			Expect(serviceErr.Message).To(ContainSubstring("Policy not found"))
		})
	})

	Describe("ListPolicies", func() {
		BeforeEach(func() {
			// Create test policies
			policies := []struct {
				id         string
				policyType v1alpha1.PolicyPolicyType
				enabled    bool
				priority   int32
			}{
				{"policy-1", v1alpha1.GLOBAL, true, 100},
				{"policy-2", v1alpha1.USER, true, 200},
				{"policy-3", v1alpha1.GLOBAL, false, 300},
				{"policy-4", v1alpha1.USER, false, 400},
			}

			for _, p := range policies {
				enabled := p.enabled
				priority := p.priority
				displayName := "Test " + p.id
				policy := v1alpha1.Policy{
					DisplayName: &displayName,
					PolicyType:  policyTypePtr(p.policyType),
					RegoCode:    strPtr("package test"),
					Enabled:     &enabled,
					Priority:    &priority,
				}
				id := p.id
				_, err := policyService.CreatePolicy(ctx, policy, &id)
				Expect(err).To(BeNil())
			}
		})

		It("should list all policies with default ordering", func() {
			result, err := policyService.ListPolicies(ctx, nil, nil, nil, nil)

			Expect(err).To(BeNil())
			Expect(result).NotTo(BeNil())
			Expect(result.Policies).To(HaveLen(4))
			// Default order is policy_type ASC, priority ASC, id ASC
			// GLOBAL policies first, then USER policies
			Expect(*result.Policies[0].Id).To(Equal("policy-1")) // GLOBAL, priority 100
			Expect(*result.Policies[1].Id).To(Equal("policy-3")) // GLOBAL, priority 300
			Expect(*result.Policies[2].Id).To(Equal("policy-2")) // USER, priority 200
			Expect(*result.Policies[3].Id).To(Equal("policy-4")) // USER, priority 400
		})

		It("should filter by policy_type=GLOBAL", func() {
			filter := "policy_type='GLOBAL'"
			result, err := policyService.ListPolicies(ctx, &filter, nil, nil, nil)

			Expect(err).To(BeNil())
			Expect(result.Policies).To(HaveLen(2))
			for _, p := range result.Policies {
				Expect(p.PolicyType).NotTo(BeNil())
				Expect(*p.PolicyType).To(Equal(v1alpha1.GLOBAL))
			}
		})

		It("should filter by policy_type=USER", func() {
			filter := "policy_type='USER'"
			result, err := policyService.ListPolicies(ctx, &filter, nil, nil, nil)

			Expect(err).To(BeNil())
			Expect(result.Policies).To(HaveLen(2))
			for _, p := range result.Policies {
				Expect(p.PolicyType).NotTo(BeNil())
				Expect(*p.PolicyType).To(Equal(v1alpha1.USER))
			}
		})

		It("should filter by enabled=true", func() {
			filter := "enabled=true"
			result, err := policyService.ListPolicies(ctx, &filter, nil, nil, nil)

			Expect(err).To(BeNil())
			Expect(result.Policies).To(HaveLen(2))
			for _, p := range result.Policies {
				Expect(*p.Enabled).To(BeTrue())
			}
		})

		It("should filter by enabled=false", func() {
			filter := "enabled=false"
			result, err := policyService.ListPolicies(ctx, &filter, nil, nil, nil)

			Expect(err).To(BeNil())
			Expect(result.Policies).To(HaveLen(2))
			for _, p := range result.Policies {
				Expect(*p.Enabled).To(BeFalse())
			}
		})

		It("should filter by combined conditions", func() {
			filter := "policy_type='GLOBAL' AND enabled=true"
			result, err := policyService.ListPolicies(ctx, &filter, nil, nil, nil)

			Expect(err).To(BeNil())
			Expect(result.Policies).To(HaveLen(1))
			Expect(*result.Policies[0].Id).To(Equal("policy-1"))
		})

		It("should order by priority desc", func() {
			orderBy := "priority desc"
			result, err := policyService.ListPolicies(ctx, nil, &orderBy, nil, nil)

			Expect(err).To(BeNil())
			Expect(result.Policies).To(HaveLen(4))
			Expect(*result.Policies[0].Id).To(Equal("policy-4"))
			Expect(*result.Policies[1].Id).To(Equal("policy-3"))
			Expect(*result.Policies[2].Id).To(Equal("policy-2"))
			Expect(*result.Policies[3].Id).To(Equal("policy-1"))
		})

		It("should support pagination", func() {
			pageSize := int32(2)
			result, err := policyService.ListPolicies(ctx, nil, nil, nil, &pageSize)

			Expect(err).To(BeNil())
			Expect(result.Policies).To(HaveLen(2))
			Expect(result.NextPageToken).NotTo(BeNil())

			// Get next page
			result2, err := policyService.ListPolicies(ctx, nil, nil, result.NextPageToken, &pageSize)

			Expect(err).To(BeNil())
			Expect(result2.Policies).To(HaveLen(2))
			Expect(result2.NextPageToken).To(BeNil()) // No more pages
		})

		It("should validate page size minimum", func() {
			pageSize := int32(0)
			_, err := policyService.ListPolicies(ctx, nil, nil, nil, &pageSize)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("Invalid page size"))
		})

		It("should validate page size maximum", func() {
			pageSize := int32(1001)
			_, err := policyService.ListPolicies(ctx, nil, nil, nil, &pageSize)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("Invalid page size"))
		})

		It("should return error for invalid filter", func() {
			filter := "invalid_field='value'"
			_, err := policyService.ListPolicies(ctx, &filter, nil, nil, nil)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
		})

		It("should return error for invalid order by", func() {
			orderBy := "invalid_field asc"
			_, err := policyService.ListPolicies(ctx, nil, &orderBy, nil, nil)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
		})
	})

	Describe("UpdatePolicy", func() {
		It("should update mutable fields (partial patch)", func() {
			// Create a policy
			clientID := "update-test"
			enabled := true
			priority := int32(100)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Original Name"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package original"),
				Enabled:     &enabled,
				Priority:    &priority,
			}
			created, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			// PATCH: only update display_name, enabled, priority, description
			newEnabled := false
			newPriority := int32(200)
			newDescription := "Updated description"
			displayName := "Updated Name"
			patch := &v1alpha1.Policy{
				DisplayName: &displayName,
				Enabled:     &newEnabled,
				Priority:    &newPriority,
				Description: &newDescription,
			}

			updated, err := policyService.UpdatePolicy(ctx, "update-test", patch)

			Expect(err).To(BeNil())
			Expect(updated.DisplayName).NotTo(BeNil())
			Expect(*updated.DisplayName).To(Equal("Updated Name"))
			Expect(*updated.Enabled).To(BeFalse())
			Expect(*updated.Priority).To(Equal(int32(200)))
			Expect(*updated.Description).To(Equal("Updated description"))
			Expect(*updated.Id).To(Equal("update-test"))                // ID unchanged
			Expect(updated.CreateTime).To(Equal(created.CreateTime))    // CreateTime unchanged
			Expect(updated.UpdateTime).NotTo(Equal(created.UpdateTime)) // UpdateTime changed
		})

		It("should validate RegoCode is non-empty when provided in patch", func() {
			clientID := "update-rego-test"
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
			}
			_, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			// Try to update with empty RegoCode in patch
			emptyRego := ""
			patch := &v1alpha1.Policy{
				RegoCode: &emptyRego,
			}

			_, err = policyService.UpdatePolicy(ctx, "update-rego-test", patch)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
		})

		It("should return NotFound error for non-existent policy", func() {
			displayName := "Test"
			patch := &v1alpha1.Policy{
				DisplayName: &displayName,
			}

			_, err := policyService.UpdatePolicy(ctx, "non-existent", patch)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeNotFound))
		})

		It("should return AlreadyExists when updating to another policy's display_name and policy_type", func() {
			regoCode := "package test"
			prioA := int32(200)
			prioB := int32(300)
			idA := "update-dn-a"
			idB := "update-dn-b"
			_, err := policyService.CreatePolicy(ctx, v1alpha1.Policy{
				DisplayName: strPtr("Name A"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    &regoCode,
				Priority:    &prioA,
			}, &idA)
			Expect(err).To(BeNil())
			_, err = policyService.CreatePolicy(ctx, v1alpha1.Policy{
				DisplayName: strPtr("Name B"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    &regoCode,
				Priority:    &prioB,
			}, &idB)
			Expect(err).To(BeNil())

			displayNameA := "Name A"
			patch := &v1alpha1.Policy{
				DisplayName: &displayNameA,
				Priority:    &prioB,
			}
			_, err = policyService.UpdatePolicy(ctx, "update-dn-b", patch)
			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeAlreadyExists))
			Expect(serviceErr.Message).To(ContainSubstring("Policy display name and policy type"))
		})

		It("should return AlreadyExists when updating to another policy's priority and policy_type", func() {
			regoCode := "package test"
			prio200 := int32(200)
			prio300 := int32(300)
			idA := "update-prio-a"
			idB := "update-prio-b"
			_, err := policyService.CreatePolicy(ctx, v1alpha1.Policy{
				DisplayName: strPtr("Policy A"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    &regoCode,
				Priority:    &prio200,
			}, &idA)
			Expect(err).To(BeNil())
			_, err = policyService.CreatePolicy(ctx, v1alpha1.Policy{
				DisplayName: strPtr("Policy B"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    &regoCode,
				Priority:    &prio300,
			}, &idB)
			Expect(err).To(BeNil())

			displayNameB := "Policy B"
			patch := &v1alpha1.Policy{
				DisplayName: &displayNameB,
				Priority:    &prio200,
			}
			_, err = policyService.UpdatePolicy(ctx, "update-prio-b", patch)
			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeAlreadyExists))
			Expect(serviceErr.Message).To(ContainSubstring("Policy priority and policy type"))
		})

		It("should reject update with priority below minimum (0)", func() {
			clientID := "update-prio-min-test"
			priority := int32(500)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}
			_, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			invalidPriority := int32(0)
			patch := &v1alpha1.Policy{
				Priority: &invalidPriority,
			}

			_, err = policyService.UpdatePolicy(ctx, "update-prio-min-test", patch)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("priority must be between 1 and 1000"))
		})

		It("should reject update with priority above maximum (1001)", func() {
			clientID := "update-prio-max-test"
			priority := int32(500)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}
			_, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			invalidPriority := int32(1001)
			patch := &v1alpha1.Policy{
				Priority: &invalidPriority,
			}

			_, err = policyService.UpdatePolicy(ctx, "update-prio-max-test", patch)

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeInvalidArgument))
			Expect(serviceErr.Message).To(ContainSubstring("priority must be between 1 and 1000"))
		})

		It("should accept update with valid priority (800)", func() {
			clientID := "update-prio-valid-test"
			priority := int32(500)
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test Policy"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
				Priority:    &priority,
			}
			_, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			newPriority := int32(800)
			patch := &v1alpha1.Policy{
				Priority: &newPriority,
			}

			updated, err := policyService.UpdatePolicy(ctx, "update-prio-valid-test", patch)

			Expect(err).To(BeNil())
			Expect(updated).NotTo(BeNil())
			Expect(*updated.Priority).To(Equal(int32(800)))
		})
	})

	Describe("DeletePolicy", func() {
		It("should delete existing policy", func() {
			// Create a policy
			clientID := "delete-test"
			policy := v1alpha1.Policy{
				DisplayName: strPtr("Test"),
				PolicyType:  policyTypePtr(v1alpha1.GLOBAL),
				RegoCode:    strPtr("package test"),
			}
			_, err := policyService.CreatePolicy(ctx, policy, &clientID)
			Expect(err).To(BeNil())

			// Delete the policy
			err = policyService.DeletePolicy(ctx, "delete-test")

			Expect(err).To(BeNil())

			// Verify it's deleted
			_, err = policyService.GetPolicy(ctx, "delete-test")
			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeNotFound))
		})

		It("should return NotFound error for non-existent policy", func() {
			err := policyService.DeletePolicy(ctx, "non-existent")

			Expect(err).NotTo(BeNil())
			serviceErr, ok := err.(*service.ServiceError)
			Expect(ok).To(BeTrue())
			Expect(serviceErr.Type).To(Equal(service.ErrorTypeNotFound))
		})
	})
})

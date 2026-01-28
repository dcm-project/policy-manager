package v1alpha1

import (
	"context"

	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/api/server"
	"github.com/dcm-project/policy-manager/internal/service"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// MockPolicyService is a mock implementation of PolicyService for testing
type MockPolicyService struct {
	CreatePolicyFn func(ctx context.Context, policy v1alpha1.Policy, clientID *string) (*v1alpha1.Policy, error)
	GetPolicyFn    func(ctx context.Context, id string) (*v1alpha1.Policy, error)
	ListPoliciesFn func(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error)
	UpdatePolicyFn func(ctx context.Context, id string, policy v1alpha1.Policy) (*v1alpha1.Policy, error)
	DeletePolicyFn func(ctx context.Context, id string) error
}

func (m *MockPolicyService) CreatePolicy(ctx context.Context, policy v1alpha1.Policy, clientID *string) (*v1alpha1.Policy, error) {
	if m.CreatePolicyFn != nil {
		return m.CreatePolicyFn(ctx, policy, clientID)
	}
	return nil, nil
}

func (m *MockPolicyService) GetPolicy(ctx context.Context, id string) (*v1alpha1.Policy, error) {
	if m.GetPolicyFn != nil {
		return m.GetPolicyFn(ctx, id)
	}
	return nil, nil
}

func (m *MockPolicyService) ListPolicies(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error) {
	if m.ListPoliciesFn != nil {
		return m.ListPoliciesFn(ctx, filter, orderBy, pageToken, pageSize)
	}
	return nil, nil
}

func (m *MockPolicyService) UpdatePolicy(ctx context.Context, id string, policy v1alpha1.Policy) (*v1alpha1.Policy, error) {
	if m.UpdatePolicyFn != nil {
		return m.UpdatePolicyFn(ctx, id, policy)
	}
	return nil, nil
}

func (m *MockPolicyService) DeletePolicy(ctx context.Context, id string) error {
	if m.DeletePolicyFn != nil {
		return m.DeletePolicyFn(ctx, id)
	}
	return nil
}

var _ = Describe("PolicyHandler", func() {
	var handler *PolicyHandler
	var mockService *MockPolicyService

	BeforeEach(func() {
		mockService = &MockPolicyService{}
		handler = NewPolicyHandler(mockService)
	})

	Describe("GetHealth", func() {
		It("should return a successful health response with correct status and path", func() {
			ctx := context.Background()
			response, err := handler.GetHealth(ctx, server.GetHealthRequestObject{})

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())

			healthResponse, ok := response.(server.GetHealth200JSONResponse)
			Expect(ok).To(BeTrue(), "response should be GetHealth200JSONResponse")

			Expect(healthResponse.Status).NotTo(BeNil())
			Expect(healthResponse.Status).To(Equal("ok"))

			Expect(healthResponse.Path).NotTo(BeNil())
			Expect(*healthResponse.Path).To(Equal("health"))
		})
	})

	Describe("CreatePolicy", func() {
		It("should return 201 on successful creation", func() {
			ctx := context.Background()
			policyID := "test-policy"
			path := "policies/test-policy"
			enabled := true
			priority := int32(500)

			mockService.CreatePolicyFn = func(ctx context.Context, policy v1alpha1.Policy, clientID *string) (*v1alpha1.Policy, error) {
				return &v1alpha1.Policy{
					Id:          &policyID,
					Path:        &path,
					DisplayName: policy.DisplayName,
					PolicyType:  policy.PolicyType,
					Enabled:     &enabled,
					Priority:    &priority,
					RegoCode:    "",
				}, nil
			}

			body := server.Policy{
				DisplayName: "Test Policy",
				PolicyType:  server.GLOBAL,
				RegoCode:    "package test",
			}

			response, err := handler.CreatePolicy(ctx, server.CreatePolicyRequestObject{
				Body: &body,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())

			createResponse, ok := response.(server.CreatePolicy201JSONResponse)
			Expect(ok).To(BeTrue(), "response should be CreatePolicy201JSONResponse")
			Expect(*createResponse.Body.Id).To(Equal("test-policy"))
		})

		It("should return 400 when body is nil", func() {
			ctx := context.Background()

			response, err := handler.CreatePolicy(ctx, server.CreatePolicyRequestObject{
				Body: nil,
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.CreatePolicy400JSONResponse)
			Expect(ok).To(BeTrue(), "response should be CreatePolicy400JSONResponse")
		})

		It("should return 409 when policy already exists", func() {
			ctx := context.Background()

			mockService.CreatePolicyFn = func(ctx context.Context, policy v1alpha1.Policy, clientID *string) (*v1alpha1.Policy, error) {
				return nil, service.NewAlreadyExistsError("Policy already exists", "Duplicate ID")
			}

			body := server.Policy{
				DisplayName: "Test Policy",
				PolicyType:  server.GLOBAL,
				RegoCode:    "package test",
			}

			response, err := handler.CreatePolicy(ctx, server.CreatePolicyRequestObject{
				Body: &body,
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.CreatePolicy409JSONResponse)
			Expect(ok).To(BeTrue(), "response should be CreatePolicy409JSONResponse")
		})
	})

	Describe("GetPolicy", func() {
		It("should return 200 with policy on success", func() {
			ctx := context.Background()
			policyID := "test-policy"
			path := "policies/test-policy"

			mockService.GetPolicyFn = func(ctx context.Context, id string) (*v1alpha1.Policy, error) {
				return &v1alpha1.Policy{
					Id:          &policyID,
					Path:        &path,
					DisplayName: "Test Policy",
					PolicyType:  v1alpha1.GLOBAL,
					RegoCode:    "",
				}, nil
			}

			response, err := handler.GetPolicy(ctx, server.GetPolicyRequestObject{
				PolicyId: "test-policy",
			})

			Expect(err).NotTo(HaveOccurred())
			policy, ok := response.(server.GetPolicy200JSONResponse)
			Expect(ok).To(BeTrue(), "response should be GetPolicy200JSONResponse")
			Expect(*policy.Id).To(Equal("test-policy"))
		})

		It("should return 404 when policy not found", func() {
			ctx := context.Background()

			mockService.GetPolicyFn = func(ctx context.Context, id string) (*v1alpha1.Policy, error) {
				return nil, service.NewNotFoundError("Policy not found", "Not found")
			}

			response, err := handler.GetPolicy(ctx, server.GetPolicyRequestObject{
				PolicyId: "non-existent",
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.GetPolicy404JSONResponse)
			Expect(ok).To(BeTrue(), "response should be GetPolicy404JSONResponse")
		})
	})

	Describe("ListPolicies", func() {
		It("should return 200 with list of policies", func() {
			ctx := context.Background()
			policyID1 := "policy-1"
			policyID2 := "policy-2"
			path1 := "policies/policy-1"
			path2 := "policies/policy-2"

			mockService.ListPoliciesFn = func(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error) {
				return &v1alpha1.ListPoliciesResponse{
					Policies: []v1alpha1.Policy{
						{
							Id:          &policyID1,
							Path:        &path1,
							DisplayName: "Policy 1",
							PolicyType:  v1alpha1.GLOBAL,
							RegoCode:    "",
						},
						{
							Id:          &policyID2,
							Path:        &path2,
							DisplayName: "Policy 2",
							PolicyType:  v1alpha1.USER,
							RegoCode:    "",
						},
					},
					NextPageToken: nil,
				}, nil
			}

			response, err := handler.ListPolicies(ctx, server.ListPoliciesRequestObject{
				Params: server.ListPoliciesParams{},
			})

			Expect(err).NotTo(HaveOccurred())
			listResponse, ok := response.(server.ListPolicies200JSONResponse)
			Expect(ok).To(BeTrue(), "response should be ListPolicies200JSONResponse")
			Expect(listResponse.Policies).To(HaveLen(2))
			Expect(*listResponse.Policies[0].Id).To(Equal("policy-1"))
			Expect(*listResponse.Policies[1].Id).To(Equal("policy-2"))
		})

		It("should pass filter parameter to service", func() {
			ctx := context.Background()
			filter := "policy_type='GLOBAL'"
			var receivedFilter *string

			mockService.ListPoliciesFn = func(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error) {
				receivedFilter = filter
				return &v1alpha1.ListPoliciesResponse{
					Policies: []v1alpha1.Policy{},
				}, nil
			}

			_, err := handler.ListPolicies(ctx, server.ListPoliciesRequestObject{
				Params: server.ListPoliciesParams{
					Filter: &filter,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(receivedFilter).NotTo(BeNil())
			Expect(*receivedFilter).To(Equal("policy_type='GLOBAL'"))
		})

		It("should pass pagination parameters to service", func() {
			ctx := context.Background()
			pageToken := "token123"
			pageSize := int32(10)
			var receivedPageToken *string
			var receivedPageSize *int32

			mockService.ListPoliciesFn = func(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error) {
				receivedPageToken = pageToken
				receivedPageSize = pageSize
				return &v1alpha1.ListPoliciesResponse{
					Policies: []v1alpha1.Policy{},
				}, nil
			}

			_, err := handler.ListPolicies(ctx, server.ListPoliciesRequestObject{
				Params: server.ListPoliciesParams{
					PageToken:   &pageToken,
					MaxPageSize: &pageSize,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(receivedPageToken).NotTo(BeNil())
			Expect(*receivedPageToken).To(Equal("token123"))
			Expect(receivedPageSize).NotTo(BeNil())
			Expect(*receivedPageSize).To(Equal(int32(10)))
		})

		It("should return 400 for invalid filter", func() {
			ctx := context.Background()

			mockService.ListPoliciesFn = func(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error) {
				return nil, service.NewInvalidArgumentError("Invalid filter", "Bad filter expression")
			}

			filter := "invalid_field='value'"
			response, err := handler.ListPolicies(ctx, server.ListPoliciesRequestObject{
				Params: server.ListPoliciesParams{
					Filter: &filter,
				},
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.ListPolicies400JSONResponse)
			Expect(ok).To(BeTrue(), "response should be ListPolicies400JSONResponse")
		})
	})

	Describe("ApplyPolicy", func() {
		It("should return 200 on successful update", func() {
			ctx := context.Background()
			policyID := "test-policy"
			path := "policies/test-policy"
			enabled := false
			priority := int32(200)

			mockService.UpdatePolicyFn = func(ctx context.Context, id string, policy v1alpha1.Policy) (*v1alpha1.Policy, error) {
				return &v1alpha1.Policy{
					Id:          &policyID,
					Path:        &path,
					DisplayName: policy.DisplayName,
					PolicyType:  policy.PolicyType,
					Enabled:     &enabled,
					Priority:    &priority,
					RegoCode:    "",
				}, nil
			}

			body := server.Policy{
				DisplayName: "Updated Policy",
				PolicyType:  server.GLOBAL,
				RegoCode:    "package updated",
			}

			response, err := handler.ApplyPolicy(ctx, server.ApplyPolicyRequestObject{
				PolicyId: "test-policy",
				Body:     &body,
			})

			Expect(err).NotTo(HaveOccurred())
			updateResponse, ok := response.(server.ApplyPolicy200JSONResponse)
			Expect(ok).To(BeTrue(), "response should be ApplyPolicy200JSONResponse")
			Expect(*updateResponse.Id).To(Equal("test-policy"))
			Expect(updateResponse.DisplayName).To(Equal("Updated Policy"))
		})

		It("should return 400 when body is nil", func() {
			ctx := context.Background()

			response, err := handler.ApplyPolicy(ctx, server.ApplyPolicyRequestObject{
				PolicyId: "test-policy",
				Body:     nil,
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.ApplyPolicy400JSONResponse)
			Expect(ok).To(BeTrue(), "response should be ApplyPolicy400JSONResponse")
		})

		It("should return 400 for failed precondition (immutable field change)", func() {
			ctx := context.Background()

			mockService.UpdatePolicyFn = func(ctx context.Context, id string, policy v1alpha1.Policy) (*v1alpha1.Policy, error) {
				return nil, service.NewFailedPreconditionError("Cannot change policy_type", "Field is immutable")
			}

			body := server.Policy{
				DisplayName: "Updated Policy",
				PolicyType:  server.USER, // Trying to change policy_type
				RegoCode:    "package test",
			}

			response, err := handler.ApplyPolicy(ctx, server.ApplyPolicyRequestObject{
				PolicyId: "test-policy",
				Body:     &body,
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.ApplyPolicy400JSONResponse)
			Expect(ok).To(BeTrue(), "response should be ApplyPolicy400JSONResponse")
		})

		It("should return 404 when policy not found", func() {
			ctx := context.Background()

			mockService.UpdatePolicyFn = func(ctx context.Context, id string, policy v1alpha1.Policy) (*v1alpha1.Policy, error) {
				return nil, service.NewNotFoundError("Policy not found", "Not found")
			}

			body := server.Policy{
				DisplayName: "Updated Policy",
				PolicyType:  server.GLOBAL,
				RegoCode:    "package test",
			}

			response, err := handler.ApplyPolicy(ctx, server.ApplyPolicyRequestObject{
				PolicyId: "non-existent",
				Body:     &body,
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.ApplyPolicy404JSONResponse)
			Expect(ok).To(BeTrue(), "response should be ApplyPolicy404JSONResponse")
		})
	})

	Describe("DeletePolicy", func() {
		It("should return 204 on successful deletion", func() {
			ctx := context.Background()

			mockService.DeletePolicyFn = func(ctx context.Context, id string) error {
				return nil
			}

			response, err := handler.DeletePolicy(ctx, server.DeletePolicyRequestObject{
				PolicyId: "test-policy",
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.DeletePolicy204Response)
			Expect(ok).To(BeTrue(), "response should be DeletePolicy204Response")
		})

		It("should return 404 when policy not found", func() {
			ctx := context.Background()

			mockService.DeletePolicyFn = func(ctx context.Context, id string) error {
				return service.NewNotFoundError("Policy not found", "Not found")
			}

			response, err := handler.DeletePolicy(ctx, server.DeletePolicyRequestObject{
				PolicyId: "non-existent",
			})

			Expect(err).NotTo(HaveOccurred())
			_, ok := response.(server.DeletePolicy404JSONResponse)
			Expect(ok).To(BeTrue(), "response should be DeletePolicy404JSONResponse")
		})
	})
})

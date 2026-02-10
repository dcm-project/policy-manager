//go:build e2e

package e2e_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	engineapi "github.com/dcm-project/policy-manager/api/v1alpha1/engine"
	"github.com/dcm-project/policy-manager/pkg/engineclient"
)

var _ = Describe("Engine API - Stub Implementation", func() {
	var (
		engineClient *engineclient.ClientWithResponses
		ctx          context.Context
	)

	BeforeEach(func() {
		engineURL := getEnvOrDefault("ENGINE_API_URL", "http://localhost:8081/api/v1alpha1")
		var err error
		engineClient, err = engineclient.NewClientWithResponses(engineURL)
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
	})

	Describe("POST /policies:evaluateRequest", func() {
		It("should return 501 Not Implemented", func() {
			request := engineapi.EvaluateRequest{
				ServiceInstance: engineapi.ServiceInstance{
					Spec: map[string]interface{}{
						"test": "data",
					},
				},
			}

			resp, err := engineClient.EvaluateRequestWithResponse(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			// Should return 500 response with 501 status
			Expect(resp.StatusCode()).To(Equal(http.StatusInternalServerError))
			Expect(resp.JSON500).NotTo(BeNil())
			Expect(resp.JSON500.Status).To(Equal(int32(501)))
			Expect(resp.JSON500.Title).To(Equal("Not Implemented"))
			Expect(resp.JSON500.Detail).NotTo(BeNil())
			Expect(*resp.JSON500.Detail).To(ContainSubstring("not yet implemented"))
		})

		It("should be reachable and return proper error format", func() {
			request := engineapi.EvaluateRequest{
				ServiceInstance: engineapi.ServiceInstance{
					Spec: map[string]interface{}{},
				},
			}

			resp, err := engineClient.EvaluateRequestWithResponse(ctx, request)
			Expect(err).NotTo(HaveOccurred())

			// Verify RFC 7807 format
			Expect(resp.JSON500).NotTo(BeNil())
			Expect(resp.JSON500.Type).To(Equal("about:blank"))
			Expect(resp.JSON500.Instance).NotTo(BeNil())
			Expect(*resp.JSON500.Instance).To(Equal("/api/v1alpha1/policies:evaluateRequest"))
		})
	})
})

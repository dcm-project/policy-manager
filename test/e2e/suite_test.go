//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/dcm-project/policy-manager/internal/opa"
	"github.com/dcm-project/policy-manager/pkg/client"
)

var (
	apiClient *client.ClientWithResponses
	opaClient opa.Client
	ctx       context.Context
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()

	// Get API URL from environment or use default
	apiURL := getEnvOrDefault("API_URL", "http://localhost:8080/api/v1alpha1")
	opaURL := getEnvOrDefault("OPA_URL", "http://localhost:8181")

	// Initialize API client
	var err error
	apiClient, err = client.NewClientWithResponses(apiURL)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API client")

	// Initialize OPA client for direct verification in e2e tests
	opaClient = opa.NewClient(opaURL, 10*time.Second)

	// Verify health endpoint
	Eventually(func() error {
		resp, err := http.Get(apiURL + "/health")
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("health check returned %d", resp.StatusCode)
		}
		return nil
	}, 30*time.Second, 1*time.Second).Should(Succeed(), "Service should be healthy")
})

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func ptr[T any](v T) *T {
	return &v
}

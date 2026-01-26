package v1alpha1

import (
	"context"
	"fmt"

	"github.com/dcm-project/policy-manager/internal/api/server"
)

const (
	ApiPrefix = "/api/v1alpha1/"
)

type PolicyHandler struct {
}

func NewPolicyHandler() *PolicyHandler {
	return &PolicyHandler{}
}

// (GET /health)
func (s *PolicyHandler) GetHealth(ctx context.Context, request server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	status := "ok"
	path := fmt.Sprintf("%shealth", ApiPrefix)
	return server.GetHealth200JSONResponse{
		Status: status,
		Path:   &path,
	}, nil
}

func (s *PolicyHandler) ApplyPolicy(ctx context.Context, request server.ApplyPolicyRequestObject) (server.ApplyPolicyResponseObject, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PolicyHandler) CreatePolicy(ctx context.Context, request server.CreatePolicyRequestObject) (server.CreatePolicyResponseObject, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PolicyHandler) DeletePolicy(ctx context.Context, request server.DeletePolicyRequestObject) (server.DeletePolicyResponseObject, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PolicyHandler) GetPolicy(ctx context.Context, request server.GetPolicyRequestObject) (server.GetPolicyResponseObject, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *PolicyHandler) ListPolicies(ctx context.Context, request server.ListPoliciesRequestObject) (server.ListPoliciesResponseObject, error) {
	return nil, fmt.Errorf("not implemented")
}
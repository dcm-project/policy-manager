package engine

import (
	"context"

	engineserver "github.com/dcm-project/policy-manager/internal/api/engine"
)

// Handler implements the engine API with stub methods
type Handler struct{}

var _ engineserver.StrictServerInterface = (*Handler)(nil)

// NewHandler creates a new engine handler
func NewHandler() *Handler {
	return &Handler{}
}

// EvaluateRequest is a stub that returns 501 Not Implemented
func (h *Handler) EvaluateRequest(ctx context.Context, request engineserver.EvaluateRequestRequestObject) (engineserver.EvaluateRequestResponseObject, error) {
	// Stub: Return 501 Not Implemented
	detail := "Policy evaluation is not yet implemented"
	instance := "/api/v1alpha1/policies:evaluateRequest"

	return engineserver.EvaluateRequest500JSONResponse{
		InternalServerErrorJSONResponse: engineserver.InternalServerErrorJSONResponse{
			Type:     "about:blank",
			Status:   501,
			Title:    "Not Implemented",
			Detail:   &detail,
			Instance: &instance,
		},
	}, nil
}

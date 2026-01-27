package v1alpha1

import (
	"context"
	"fmt"

	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/api/server"
	"github.com/dcm-project/policy-manager/internal/service"
)

const (
	ApiPrefix = "/api/v1alpha1/"
)

type PolicyHandler struct {
	service service.PolicyService
}

// Ensure PolicyHandler implements StrictServerInterface
var _ server.StrictServerInterface = (*PolicyHandler)(nil)

func NewPolicyHandler(service service.PolicyService) *PolicyHandler {
	return &PolicyHandler{
		service: service,
	}
}

// (GET /health)
func (h *PolicyHandler) GetHealth(ctx context.Context, request server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	status := "ok"
	path := fmt.Sprintf("%shealth", ApiPrefix)
	return server.GetHealth200JSONResponse{
		Status: status,
		Path:   &path,
	}, nil
}

// (POST /policies)
func (h *PolicyHandler) CreatePolicy(ctx context.Context, request server.CreatePolicyRequestObject) (server.CreatePolicyResponseObject, error) {
	if request.Body == nil {
		return server.CreatePolicy400JSONResponse{
			BadRequestJSONResponse: badRequestResponse(buildErrorResponse(
				400,
				v1alpha1.INVALIDARGUMENT,
				"Invalid request body",
				strPtr("Request body is required"),
			)),
		}, nil
	}

	// Convert server.Policy to v1alpha1.Policy
	v1Alpha1Policy := policyServerToV1Alpha1(*request.Body)

	// Call service to create policy
	created, err := h.service.CreatePolicy(ctx, v1Alpha1Policy, request.Params.Id)
	if err != nil {
		return h.handleCreatePolicyError(err, request), nil
	}

	// Build Location header
	location := fmt.Sprintf("%spolicies/%s", ApiPrefix, *created.Id)

	// Convert back to server.Policy
	return server.CreatePolicy201JSONResponse{
		Body: policyV1Alpha1ToServer(*created),
		Headers: server.CreatePolicy201ResponseHeaders{
			Location: location,
		},
	}, nil
}

// (GET /policies/{policyId})
func (h *PolicyHandler) GetPolicy(ctx context.Context, request server.GetPolicyRequestObject) (server.GetPolicyResponseObject, error) {
	// Call service to get policy
	policy, err := h.service.GetPolicy(ctx, request.PolicyId)
	if err != nil {
		return h.handleGetPolicyError(err, request), nil
	}

	return server.GetPolicy200JSONResponse(policyV1Alpha1ToServer(*policy)), nil
}

// (GET /policies)
func (h *PolicyHandler) ListPolicies(ctx context.Context, request server.ListPoliciesRequestObject) (server.ListPoliciesResponseObject, error) {
	// Extract parameters with defaults handled by service
	result, err := h.service.ListPolicies(
		ctx,
		request.Params.Filter,
		request.Params.OrderBy,
		request.Params.PageToken,
		request.Params.MaxPageSize,
	)
	if err != nil {
		return h.handleListPoliciesError(err, request), nil
	}

	return server.ListPolicies200JSONResponse(listResponseV1Alpha1ToServer(*result)), nil
}

// (PUT /policies/{policyId})
func (h *PolicyHandler) ApplyPolicy(ctx context.Context, request server.ApplyPolicyRequestObject) (server.ApplyPolicyResponseObject, error) {
	if request.Body == nil {
		return server.ApplyPolicy400JSONResponse{
			BadRequestJSONResponse: badRequestResponse(buildErrorResponse(
				400,
				v1alpha1.INVALIDARGUMENT,
				"Invalid request body",
				strPtr("Request body is required"),
			)),
		}, nil
	}

	// Convert server.Policy to v1alpha1.Policy
	v1Alpha1Policy := policyServerToV1Alpha1(*request.Body)

	// Call service to update policy
	updated, err := h.service.UpdatePolicy(ctx, request.PolicyId, v1Alpha1Policy)
	if err != nil {
		return h.handleApplyPolicyError(err, request), nil
	}

	return server.ApplyPolicy200JSONResponse(policyV1Alpha1ToServer(*updated)), nil
}

// (DELETE /policies/{policyId})
func (h *PolicyHandler) DeletePolicy(ctx context.Context, request server.DeletePolicyRequestObject) (server.DeletePolicyResponseObject, error) {
	// Call service to delete policy
	err := h.service.DeletePolicy(ctx, request.PolicyId)
	if err != nil {
		return h.handleDeletePolicyError(err, request), nil
	}

	return server.DeletePolicy204Response{}, nil
}

// Error handling helpers

func (h *PolicyHandler) handleCreatePolicyError(err error, request server.CreatePolicyRequestObject) server.CreatePolicyResponseObject {
	serviceErr, ok := err.(*service.ServiceError)
	if !ok {
		return server.CreatePolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(err.Error()),
			)),
		}
	}

	switch serviceErr.Type {
	case service.ErrorTypeInvalidArgument:
		return server.CreatePolicy400JSONResponse{
			BadRequestJSONResponse: badRequestResponse(buildErrorResponse(
				400,
				v1alpha1.INVALIDARGUMENT,
				serviceErr.Message,
				strPtr(serviceErr.Detail),
			)),
		}
	case service.ErrorTypeAlreadyExists:
		return server.CreatePolicy409JSONResponse{
			AlreadyExistsJSONResponse: alreadyExistsResponse(buildErrorResponse(
				409,
				v1alpha1.ALREADYEXISTS,
				serviceErr.Message,
				strPtr(serviceErr.Detail),
			)),
		}
	default:
		return server.CreatePolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(serviceErr.Detail),
			)),
		}
	}
}

func (h *PolicyHandler) handleGetPolicyError(err error, request server.GetPolicyRequestObject) server.GetPolicyResponseObject {
	serviceErr, ok := err.(*service.ServiceError)
	if !ok {
		return server.GetPolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(err.Error()),
			)),
		}
	}

	switch serviceErr.Type {
	case service.ErrorTypeNotFound:
		return server.GetPolicy404JSONResponse{
			NotFoundJSONResponse: notFoundResponse(buildErrorResponse(
				404,
				v1alpha1.NOTFOUND,
				serviceErr.Message,
				strPtr(serviceErr.Detail),
			)),
		}
	default:
		return server.GetPolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(serviceErr.Detail),
			)),
		}
	}
}

func (h *PolicyHandler) handleListPoliciesError(err error, request server.ListPoliciesRequestObject) server.ListPoliciesResponseObject {
	serviceErr, ok := err.(*service.ServiceError)
	if !ok {
		return server.ListPolicies500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(err.Error()),
			)),
		}
	}

	switch serviceErr.Type {
	case service.ErrorTypeInvalidArgument:
		return server.ListPolicies400JSONResponse{
			BadRequestJSONResponse: badRequestResponse(buildErrorResponse(
				400,
				v1alpha1.INVALIDARGUMENT,
				serviceErr.Message,
				strPtr(serviceErr.Detail),
			)),
		}
	default:
		return server.ListPolicies500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(serviceErr.Detail),
			)),
		}
	}
}

func (h *PolicyHandler) handleApplyPolicyError(err error, request server.ApplyPolicyRequestObject) server.ApplyPolicyResponseObject {
	serviceErr, ok := err.(*service.ServiceError)
	if !ok {
		return server.ApplyPolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(err.Error()),
			)),
		}
	}

	switch serviceErr.Type {
	case service.ErrorTypeInvalidArgument, service.ErrorTypeFailedPrecondition:
		return server.ApplyPolicy400JSONResponse{
			BadRequestJSONResponse: badRequestResponse(buildErrorResponse(
				400,
				v1alpha1.INVALIDARGUMENT,
				serviceErr.Message,
				strPtr(serviceErr.Detail),
			)),
		}
	case service.ErrorTypeNotFound:
		return server.ApplyPolicy404JSONResponse{
			NotFoundJSONResponse: notFoundResponse(buildErrorResponse(
				404,
				v1alpha1.NOTFOUND,
				serviceErr.Message,
				strPtr(serviceErr.Detail),
			)),
		}
	default:
		return server.ApplyPolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(serviceErr.Detail),
			)),
		}
	}
}

func (h *PolicyHandler) handleDeletePolicyError(err error, request server.DeletePolicyRequestObject) server.DeletePolicyResponseObject {
	serviceErr, ok := err.(*service.ServiceError)
	if !ok {
		return server.DeletePolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(err.Error()),
			)),
		}
	}

	switch serviceErr.Type {
	case service.ErrorTypeNotFound:
		return server.DeletePolicy404JSONResponse{
			NotFoundJSONResponse: notFoundResponse(buildErrorResponse(
				404,
				v1alpha1.NOTFOUND,
				serviceErr.Message,
				strPtr(serviceErr.Detail),
			)),
		}
	default:
		return server.DeletePolicy500JSONResponse{
			InternalServerErrorJSONResponse: internalErrorResponse(buildErrorResponse(
				500,
				v1alpha1.INTERNAL,
				"Internal server error",
				strPtr(serviceErr.Detail),
			)),
		}
	}
}

// buildErrorResponse builds an RFC 7807 error response
func buildErrorResponse(status int32, errorType v1alpha1.ErrorType, title string, detail *string) v1alpha1.Error {
	return v1alpha1.Error{
		Status: status,
		Type:   errorType,
		Title:  title,
		Detail: detail,
	}
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}

// Type conversion helpers between server and v1alpha1 packages

func policyServerToV1Alpha1(p server.Policy) v1alpha1.Policy {
	return v1alpha1.Policy{
		CreateTime:    p.CreateTime,
		Description:   p.Description,
		DisplayName:   p.DisplayName,
		Enabled:       p.Enabled,
		Id:            p.Id,
		LabelSelector: p.LabelSelector,
		Path:          p.Path,
		PolicyType:    v1alpha1.PolicyPolicyType(p.PolicyType),
		Priority:      p.Priority,
		RegoCode:      p.RegoCode,
		UpdateTime:    p.UpdateTime,
	}
}

func policyV1Alpha1ToServer(p v1alpha1.Policy) server.Policy {
	return server.Policy{
		CreateTime:    p.CreateTime,
		Description:   p.Description,
		DisplayName:   p.DisplayName,
		Enabled:       p.Enabled,
		Id:            p.Id,
		LabelSelector: p.LabelSelector,
		Path:          p.Path,
		PolicyType:    server.PolicyPolicyType(p.PolicyType),
		Priority:      p.Priority,
		RegoCode:      p.RegoCode,
		UpdateTime:    p.UpdateTime,
	}
}

func listResponseV1Alpha1ToServer(r v1alpha1.ListPoliciesResponse) server.ListPoliciesResponse {
	policies := make([]server.Policy, len(r.Policies))
	for i, p := range r.Policies {
		policies[i] = policyV1Alpha1ToServer(p)
	}
	return server.ListPoliciesResponse{
		NextPageToken: r.NextPageToken,
		Policies:      policies,
	}
}

func serverErrorFromV1Alpha1(e v1alpha1.Error) server.Error {
	return server.Error{
		Detail:   e.Detail,
		Instance: e.Instance,
		Status:   e.Status,
		Title:    e.Title,
		Type:     server.ErrorType(e.Type),
	}
}

// Typed error response helpers
func badRequestResponse(e v1alpha1.Error) server.BadRequestJSONResponse {
	return server.BadRequestJSONResponse(serverErrorFromV1Alpha1(e))
}

func notFoundResponse(e v1alpha1.Error) server.NotFoundJSONResponse {
	return server.NotFoundJSONResponse(serverErrorFromV1Alpha1(e))
}

func alreadyExistsResponse(e v1alpha1.Error) server.AlreadyExistsJSONResponse {
	return server.AlreadyExistsJSONResponse(serverErrorFromV1Alpha1(e))
}

func internalErrorResponse(e v1alpha1.Error) server.InternalServerErrorJSONResponse {
	return server.InternalServerErrorJSONResponse(serverErrorFromV1Alpha1(e))
}

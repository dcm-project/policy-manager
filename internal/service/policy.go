package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/dcm-project/policy-manager/api/v1alpha1"
	"github.com/dcm-project/policy-manager/internal/store"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	// AEP-122 compliant ID format: 1-63 chars, start with lowercase letter,
	// contain only lowercase letters, numbers, and hyphens, end with letter or number
	idPattern = regexp.MustCompile(`^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// PolicyService defines the interface for policy business logic operations.
type PolicyService interface {
	CreatePolicy(ctx context.Context, policy v1alpha1.Policy, clientID *string) (*v1alpha1.Policy, error)
	GetPolicy(ctx context.Context, id string) (*v1alpha1.Policy, error)
	ListPolicies(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error)
	UpdatePolicy(ctx context.Context, id string, patch *v1alpha1.PolicyUpdate) (*v1alpha1.Policy, error)
	DeletePolicy(ctx context.Context, id string) error
}

// PolicyServiceImpl implements the PolicyService interface.
type PolicyServiceImpl struct {
	store store.Store
}

var _ PolicyService = (*PolicyServiceImpl)(nil)

// NewPolicyService creates a new PolicyService instance.
func NewPolicyService(store store.Store) *PolicyServiceImpl {
	return &PolicyServiceImpl{
		store: store,
	}
}

// CreatePolicy creates a new policy resource.
func (s *PolicyServiceImpl) CreatePolicy(ctx context.Context, policy v1alpha1.Policy, clientID *string) (*v1alpha1.Policy, error) {
	// Validate RegoCode is present and non-empty
	if strings.TrimSpace(policy.RegoCode) == "" {
		return nil, NewInvalidArgumentError(
			"RegoCode is required",
			"The rego_code field must be present and non-empty",
		)
	}

	// Determine the policy ID (client-specified or server-generated)
	var policyID string
	if clientID != nil && *clientID != "" {
		policyID = *clientID
		// Validate ID format (AEP-122 compliant) only for client-specified IDs
		if !idPattern.MatchString(policyID) {
			return nil, NewInvalidArgumentError(
				"Invalid policy ID format",
				fmt.Sprintf("Policy ID '%s' does not match required format: 1-63 characters, start with lowercase letter, contain only lowercase letters, numbers, and hyphens, end with letter or number", policyID),
			)
		}
	} else {
		// Generate UUID for server-assigned ID
		policyID = uuid.New().String()
	}

	// Convert API model to DB model (strips RegoCode)
	dbPolicy := APIToDBModel(policy, policyID)

	// Create policy in store
	created, err := s.store.Policy().Create(ctx, dbPolicy)
	if err != nil {
		// Check for duplicate display_name+policy_type or priority+policy_type
		if errors.Is(err, store.ErrDisplayNamePolicyTypeTaken) {
			return nil, NewAlreadyExistsError(
				"A policy with this display_name and policy_type already exists",
				"The combination of display_name and policy_type must be unique",
			)
		}
		if errors.Is(err, store.ErrPriorityPolicyTypeTaken) {
			return nil, NewAlreadyExistsError(
				"A policy with this priority and policy_type already exists",
				"The combination of priority and policy_type must be unique",
			)
		}
		// Check for duplicate ID error (GORM unique constraint violation)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") ||
			strings.Contains(err.Error(), "duplicate key") {
			return nil, NewAlreadyExistsError(
				"Policy already exists",
				fmt.Sprintf("A policy with ID '%s' already exists", policyID),
			)
		}
		if errors.Is(err, store.ErrPolicyIDTaken) {
			return nil, NewAlreadyExistsError(
				"Policy already exists",
				fmt.Sprintf("A policy with ID '%s' already exists", policyID),
			)
		}
		return nil, NewInternalError("Failed to create policy", err.Error(), err)
	}

	// Convert back to API model with empty RegoCode and set Path
	apiPolicy := DBToAPIModel(created)

	return &apiPolicy, nil
}

// GetPolicy retrieves a policy by ID.
func (s *PolicyServiceImpl) GetPolicy(ctx context.Context, id string) (*v1alpha1.Policy, error) {
	// Get policy from store
	dbPolicy, err := s.store.Policy().Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrPolicyNotFound) {
			return nil, NewNotFoundError(
				"Policy not found",
				fmt.Sprintf("Policy with ID '%s' does not exist", id),
			)
		}
		return nil, NewInternalError("Failed to get policy", err.Error(), err)
	}

	// Convert to API model with empty RegoCode
	apiPolicy := DBToAPIModel(dbPolicy)

	return &apiPolicy, nil
}

// ListPolicies lists policies with optional filtering, ordering, and pagination.
func (s *PolicyServiceImpl) ListPolicies(ctx context.Context, filter *string, orderBy *string, pageToken *string, pageSize *int32) (*v1alpha1.ListPoliciesResponse, error) {
	// Parse filter expression
	var policyFilter *store.PolicyFilter
	var err error
	if filter != nil && *filter != "" {
		policyFilter, err = ParseFilter(*filter)
		if err != nil {
			return nil, err // Already a ServiceError
		}
	}

	// Parse order by parameter (ParseOrderBy handles nil/empty with default)
	orderByStr := ""
	if orderBy != nil {
		orderByStr = *orderBy
	}
	orderByStr, err = ParseOrderBy(orderByStr)
	if err != nil {
		return nil, err // Already a ServiceError
	}

	// Validate and set page size (default: 50, max: 1000)
	pageSizeInt := 50
	if pageSize != nil {
		if *pageSize < 1 {
			return nil, NewInvalidArgumentError(
				"Invalid page size",
				"Page size must be at least 1",
			)
		}
		if *pageSize > 1000 {
			return nil, NewInvalidArgumentError(
				"Invalid page size",
				"Page size must not exceed 1000",
			)
		}
		pageSizeInt = int(*pageSize)
	}

	// Build list options
	opts := &store.PolicyListOptions{
		Filter:    policyFilter,
		OrderBy:   orderByStr,
		PageToken: pageToken,
		PageSize:  pageSizeInt,
	}

	// List policies from store
	result, err := s.store.Policy().List(ctx, opts)
	if err != nil {
		return nil, NewInternalError("Failed to list policies", err.Error(), err)
	}

	// Convert all DB models to API models
	apiPolicies := make([]v1alpha1.Policy, len(result.Policies))
	for i, dbPolicy := range result.Policies {
		apiPolicies[i] = DBToAPIModel(&dbPolicy)
	}

	// Build response
	response := &v1alpha1.ListPoliciesResponse{
		Policies: apiPolicies,
	}

	if result.NextPageToken != "" {
		response.NextPageToken = &result.NextPageToken
	}

	return response, nil
}

// MergePolicyUpdateOntoPolicy merges a PATCH body onto an existing policy per RFC 7396.
// Only non-nil fields in patch are applied. policy_type is immutable and not in PolicyUpdate.
func MergePolicyUpdateOntoPolicy(patch *v1alpha1.PolicyUpdate, existing v1alpha1.Policy) v1alpha1.Policy {
	merged := existing
	if patch == nil {
		return merged
	}
	if patch.DisplayName != nil {
		merged.DisplayName = *patch.DisplayName
	}
	if patch.Description != nil {
		merged.Description = patch.Description
	}
	if patch.Enabled != nil {
		merged.Enabled = patch.Enabled
	}
	if patch.LabelSelector != nil {
		merged.LabelSelector = patch.LabelSelector
	}
	if patch.Priority != nil {
		merged.Priority = patch.Priority
	}
	if patch.RegoCode != nil {
		merged.RegoCode = *patch.RegoCode
	}
	return merged
}

// UpdatePolicy updates an existing policy using partial merge (PATCH).
func (s *PolicyServiceImpl) UpdatePolicy(ctx context.Context, id string, patch *v1alpha1.PolicyUpdate) (*v1alpha1.Policy, error) {
	// Validate RegoCode when present in patch (RegoCode is not stored in DB, so omit = leave as-is)
	if patch != nil && patch.RegoCode != nil && strings.TrimSpace(*patch.RegoCode) == "" {
		return nil, NewInvalidArgumentError(
			"RegoCode cannot be empty",
			"When rego_code is provided in the patch it must be non-empty",
		)
	}

	// Get existing policy
	existingDB, err := s.store.Policy().Get(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrPolicyNotFound) {
			return nil, NewNotFoundError(
				"Policy not found",
				fmt.Sprintf("Policy with ID '%s' does not exist", id),
			)
		}
		return nil, NewInternalError("Failed to get existing policy", err.Error(), err)
	}
	existing := DBToAPIModel(existingDB)

	// Merge patch onto existing (policy_type in patch is ignored)
	merged := MergePolicyUpdateOntoPolicy(patch, existing)

	// Convert API model to DB model and update store
	dbPolicy := APIToDBModel(merged, id)
	updated, err := s.store.Policy().Update(ctx, dbPolicy)
	if err != nil {
		if errors.Is(err, store.ErrDisplayNamePolicyTypeTaken) {
			return nil, NewAlreadyExistsError(
				"A policy with this display_name and policy_type already exists",
				"The combination of display_name and policy_type must be unique",
			)
		}
		if errors.Is(err, store.ErrPriorityPolicyTypeTaken) {
			return nil, NewAlreadyExistsError(
				"A policy with this priority and policy_type already exists",
				"The combination of priority and policy_type must be unique",
			)
		}
		if errors.Is(err, store.ErrPolicyNotFound) {
			return nil, NewNotFoundError(
				"Policy not found",
				fmt.Sprintf("Policy with ID '%s' does not exist", id),
			)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NewNotFoundError(
				"Policy not found",
				fmt.Sprintf("Policy with ID '%s' does not exist", id),
			)
		}
		return nil, NewInternalError("Failed to update policy", err.Error(), err)
	}

	// Convert back to API model
	apiPolicy := DBToAPIModel(updated)

	return &apiPolicy, nil
}

// DeletePolicy deletes a policy by ID.
func (s *PolicyServiceImpl) DeletePolicy(ctx context.Context, id string) error {
	// Delete policy from store
	err := s.store.Policy().Delete(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrPolicyNotFound) {
			return NewNotFoundError(
				"Policy not found",
				fmt.Sprintf("Policy with ID '%s' does not exist", id),
			)
		}
		return NewInternalError("Failed to delete policy", err.Error(), err)
	}

	return nil
}

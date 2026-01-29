package service

import (
	"fmt"
	"github.com/dcm-project/policy-manager/api/v1alpha1"
)

// ErrorType represents the type of service error
type ErrorType string

const (
	ErrorTypeInvalidArgument    ErrorType = "INVALID_ARGUMENT"
	ErrorTypeNotFound           ErrorType = "NOT_FOUND"
	ErrorTypeAlreadyExists      ErrorType = "ALREADY_EXISTS"
	ErrorTypeInternal           ErrorType = "INTERNAL"
	ErrorTypeFailedPrecondition ErrorType = "FAILED_PRECONDITION"
)

// ServiceError represents a structured error from the service layer
type ServiceError struct {
	Type    ErrorType
	Message string
	Detail  string
	Err     error
}

func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *ServiceError) Unwrap() error {
	return e.Err
}

// NewInvalidArgumentError creates a new invalid argument error
func NewInvalidArgumentError(message, detail string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeInvalidArgument,
		Message: message,
		Detail:  detail,
	}
}

func NewPolicyNotFoundError(policyID string) *ServiceError {
	return NewNotFoundError("Policy not found", fmt.Sprintf("Policy with ID '%s' does not exist", policyID))
}

// NewNotFoundError creates a new not found error
func NewNotFoundError(message, detail string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeNotFound,
		Message: message,
		Detail:  detail,
	}
}

func NewPolicyAlreadyExistsError(policyID string) *ServiceError {
	return NewAlreadyExistsError("Policy already exists", fmt.Sprintf("A policy with ID '%s' already exists", policyID))
}

func NewPolicyDisplayNamePolicyTypeTakenError(displayName string, policyType v1alpha1.PolicyPolicyType) *ServiceError {
	return NewAlreadyExistsError(
		"Policy display name and policy type already exists",
		fmt.Sprintf("A policy with display name '%s' and policy type '%s' already exists", displayName, string(policyType)),
	)
}

func NewPolicyPriorityPolicyTypeTakenError(priority int32, policyType v1alpha1.PolicyPolicyType) *ServiceError {
	return NewAlreadyExistsError(
		"Policy priority and policy type already exists",
		fmt.Sprintf("A policy with priority '%d' and policy type '%s' already exists", priority, string(policyType)),
	)
}

// NewAlreadyExistsError creates a new already exists error
func NewAlreadyExistsError(message, detail string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeAlreadyExists,
		Message: message,
		Detail:  detail,
	}
}

// NewInternalError creates a new internal error
func NewInternalError(message, detail string, err error) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeInternal,
		Message: message,
		Detail:  detail,
		Err:     err,
	}
}

// NewFailedPreconditionError creates a new failed precondition error
func NewFailedPreconditionError(message, detail string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeFailedPrecondition,
		Message: message,
		Detail:  detail,
	}
}

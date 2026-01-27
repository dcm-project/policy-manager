package service

import "fmt"

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

// NewNotFoundError creates a new not found error
func NewNotFoundError(message, detail string) *ServiceError {
	return &ServiceError{
		Type:    ErrorTypeNotFound,
		Message: message,
		Detail:  detail,
	}
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

package errors

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string
	Message    string
	Detail     string
	StatusCode int
	Err        error
}

func (e *AppError) Error() string {
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewAppError(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

func (e *AppError) WithDetail(detail string) *AppError {
	e.Detail = detail

	return e
}

func (e *AppError) WithError(err error) *AppError {
	e.Err = err

	return e
}

const (
	ErrCodeValidation        = "VALIDATION_ERROR"
	ErrCodeBadRequest        = "BAD_REQUEST"
	ErrCodeNotFound          = "NOT_FOUND"
	ErrCodeUnauthorized      = "UNAUTHORIZED"
	ErrCodeForbidden         = "FORBIDDEN"
	ErrCodeInternal          = "INTERNAL_ERROR"
	ErrCodeDatabaseError     = "DATABASE_ERROR"
	ErrCodeDuplicateEntry    = "DUPLICATE_ENTRY"
	ErrCodeThirdPartyError   = "THIRD_PARTY_ERROR"
	ErrCodeTooManyRequests   = "TOO_MANY_REQUESTS"
	ErrCodeResourceExhausted = "RESOURCE_EXHAUSTED"
)

func ValidationError(message string) *AppError {
	return NewAppError(ErrCodeValidation, message, http.StatusBadRequest)
}

func BadRequestError(message string) *AppError {
	return NewAppError(ErrCodeBadRequest, message, http.StatusBadRequest)
}

func NotFoundError(message string) *AppError {
	return NewAppError(ErrCodeNotFound, message, http.StatusNotFound)
}

func UnauthorizedError(message string) *AppError {
	return NewAppError(ErrCodeUnauthorized, message, http.StatusUnauthorized)
}

func ForbiddenError(message string) *AppError {
	return NewAppError(ErrCodeForbidden, message, http.StatusForbidden)
}

func InternalError(message string) *AppError {
	return NewAppError(ErrCodeInternal, message, http.StatusInternalServerError)
}

func DatabaseError(message string) *AppError {
	return NewAppError(ErrCodeDatabaseError, message, http.StatusInternalServerError)
}

func DuplicateEntryError(message string) *AppError {
	return NewAppError(ErrCodeDuplicateEntry, message, http.StatusConflict)
}

func ThirdPartyError(message string) *AppError {
	return NewAppError(ErrCodeThirdPartyError, message, http.StatusInternalServerError)
}

func TooManyRequestsError(message string) *AppError {
	return NewAppError(ErrCodeTooManyRequests, message, http.StatusTooManyRequests)
}

func ResourceExhaustedError(message string) *AppError {
	return NewAppError(ErrCodeResourceExhausted, message, http.StatusTooManyRequests)
}

func IsAppError(err error) (*AppError, bool) {
	var appError *AppError

	if errors.As(err, &appError) {
		return appError, true
	}

	return nil, false
}

// field validation error.
func AddValidationError(field, reason string) *AppError {
	return ValidationError(fmt.Sprintf("Invalid field '%s': %s", field, reason))
}

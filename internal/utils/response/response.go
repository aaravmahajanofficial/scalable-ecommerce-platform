package response

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/go-playground/validator/v10"
)

type APIResponse struct {
	Success bool           `json:"success"`
	Data    any            `json:"data,omitempty"`
	Error   *ErrorResponse `json:"error,omitempty"`
}

type ErrorResponse struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
}

type Response struct {
	Status string `json:"status"` // or custom status response name
	Error  string `json:"error"`  // or custom error response name
}

const (
	StatusOK    = "OK"
	StatusError = "Error"
)

// interface {} == any
func WriteJson(w http.ResponseWriter, statusCode int, data any) error {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	return json.NewEncoder(w).Encode(data) //struct to json
}

func Success(w http.ResponseWriter, statusCode int, data any) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}

	WriteJson(w, statusCode, response)
}

func Error(w http.ResponseWriter, err error) {

	var statusCode int
	var errorResponse *ErrorResponse

	if appErr, ok := errors.IsAppError(err); ok {
		statusCode = appErr.StatusCode
		errorResponse = &ErrorResponse{
			Code:    appErr.Code,
			Message: appErr.Message,
		}

		if appErr.Detail != "" {
			errorResponse.Details = []string{appErr.Detail}
		}

	} else {

		statusCode = http.StatusInternalServerError
		errorResponse = &ErrorResponse{
			Code:    errors.ErrCodeInternal,
			Message: "An unexpected error occured",
		}

	}

	response := APIResponse{
		Success: false,
		Error:   errorResponse,
	}

	WriteJson(w, statusCode, response)
}

func GeneralError(err error) Response {

	return Response{
		Status: StatusError,
		Error:  err.Error(),
	}

}

// package sends the list of errors
func ValidationError(w http.ResponseWriter, errs validator.ValidationErrors) {

	var errMsgs []string

	for _, err := range errs {

		var message string

		switch err.Tag() {
		case "required":
			message = fmt.Sprintf("Field %s is required", err.Field())
		case "email":
			message = fmt.Sprintf("Field %s must be a valid email address", err.Field())
		case "min":
			message = fmt.Sprintf("Field %s must be at least %s characters", err.Field(), err.Param())
		case "max":
			message = fmt.Sprintf("Field %s must be at most %s characters", err.Field(), err.Param())
		case "gt":
			message = fmt.Sprintf("Field %s must be greater than %s", err.Field(), err.Param())
		case "lt":
			message = fmt.Sprintf("Field %s must be less than %s", err.Field(), err.Param())
		default:
			message = fmt.Sprintf("Field %s is invalid: %s=%s", err.Field(), err.Tag(), err.Param())
		}

		errMsgs = append(errMsgs, message)

	}

	errorResponse := &ErrorResponse{
		Code:    errors.ErrCodeValidation,
		Message: "Validation failed",
		Details: errMsgs,
	}

	response := APIResponse{
		Success: false,
		Error:   errorResponse,
	}

	WriteJson(w, http.StatusBadRequest, response)

}

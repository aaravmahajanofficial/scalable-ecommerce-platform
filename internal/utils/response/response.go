package response

import (
	"encoding/json"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
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
package response

import (
	"encoding/json"
	"log/slog"
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

// interface {} == any.
func WriteJSON(w http.ResponseWriter, statusCode int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	return json.NewEncoder(w).Encode(data) // struct to json
}

func Success(w http.ResponseWriter, statusCode int, data any) {
	response := APIResponse{
		Success: true,
		Data:    data,
	}

	if err := WriteJSON(w, statusCode, response); err != nil {
		slog.Error("failed to write success response", "error", err)
	}
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
			Message: "An unexpected error occurred",
		}
	}

	response := APIResponse{
		Success: false,
		Error:   errorResponse,
	}

	if writeErr := WriteJSON(w, statusCode, response); writeErr != nil {
		slog.Error("failed to write error response", "error", writeErr, "original_error", err)
	}
}

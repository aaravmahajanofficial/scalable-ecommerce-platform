package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

func DecodeJSONBody(r *http.Request, dest any) error {

	body, err := io.ReadAll(r.Body)

	if err != nil {
		slog.Error("Failed to read request body",
			slog.String("error", err.Error()),
			slog.String("endpoint", r.URL.Path),
		)
		return fmt.Errorf("failed to read request body: %w", err)
	}

	defer r.Body.Close()

	if len(body) == 0 {
		slog.Warn("Empty request body", slog.String("endpoint", r.URL.Path))
		return errors.New("request body cannot be empty")
	}

	if err := json.Unmarshal(body, dest); err != nil {
		slog.Error("Failed to parse request JSON",
			slog.String("error", err.Error()),
			slog.String("endpoint", r.URL.Path),
		)
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	return nil
}

func ValidateStruct(validate *validator.Validate, data any) error {
	if err := validate.Struct(data); err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			slog.Warn("User input validation failed",
				slog.String("error", validationErrs.Error()),
			)
			return fmt.Errorf("validation error: %w", validationErrs)

		} else {
			slog.Error("Unexpected validation error", slog.String("error", err.Error()))
			return fmt.Errorf("unexpected validation error: %w", validationErrs)
		}

	}
	return nil
}

func ParseAndValidate(r *http.Request, w http.ResponseWriter, dest any, validate *validator.Validate) bool {

	if err := DecodeJSONBody(r, dest); err != nil {
		slog.Warn("Invalid request", slog.String("error", err.Error()))
		response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
		return false
	}

	if err := ValidateStruct(validate, dest); err != nil {
		slog.Warn("Validation failed", slog.String("error", err.Error()))
		response.WriteJson(w, http.StatusBadRequest, response.GeneralError(errors.New("invalid input data")))
		return false
	}

	return true

}

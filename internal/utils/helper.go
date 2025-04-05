package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
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
		return errors.BadRequestError("Failed to read request body").WithError(err)
	}

	defer r.Body.Close()

	if len(body) == 0 {
		slog.Warn("Empty request body", slog.String("endpoint", r.URL.Path))
		return errors.BadRequestError("Request body cannot be empty").WithError(err)
	}

	if err := json.Unmarshal(body, dest); err != nil {
		slog.Error("Failed to parse request JSON",
			slog.String("error", err.Error()),
			slog.String("endpoint", r.URL.Path),
		)
		return errors.BadRequestError("Invalid JSON format").WithError(err)
	}

	return nil
}

func ValidateStruct(validate *validator.Validate, data any) error {
	if err := validate.Struct(data); err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			slog.Warn("User input validation failed",
				slog.String("error", validationErrs.Error()),
			)

			var details []string
			for _, verr := range validationErrs {
				details = append(details, formatValidationError(verr))
			}

			return errors.ValidationError("Validation Failed").WithDetail(fmt.Sprintf("%v", details))

		} else {
			slog.Error("Unexpected validation error", slog.String("error", err.Error()))
			return errors.InternalError("Unexpected validation error").WithError(err)
		}

	}
	return nil
}

func ParseAndValidate(r *http.Request, w http.ResponseWriter, dest any, validate *validator.Validate) bool {

	if err := DecodeJSONBody(r, dest); err != nil {
		response.Error(w, err)
		return false
	}

	if err := ValidateStruct(validate, dest); err != nil {
		response.Error(w, err)
		return false
	}

	return true
}

func formatValidationError(err validator.FieldError) string {

	switch err.Tag() {
	case "required":
		return fmt.Sprintf("Field %s is required", err.Field())
	case "email":
		return fmt.Sprintf("Field %s must be a valid email address", err.Field())
	case "min":
		return fmt.Sprintf("Field %s must be at least %s characters", err.Field(), err.Param())
	case "max":
		return fmt.Sprintf("Field %s must be at most %s characters", err.Field(), err.Param())
	case "gt":
		return fmt.Sprintf("Field %s must be greater than %s", err.Field(), err.Param())
	case "lt":
		return fmt.Sprintf("Field %s must be less than %s", err.Field(), err.Param())
	default:
		return fmt.Sprintf("Field %s is invalid: %s=%s", err.Field(), err.Tag(), err.Param())
	}
}

func ParseID(r *http.Request, paramName string) (int64, error) {
	idStr := r.PathValue(paramName)
	id, err := strconv.ParseInt(idStr, 10, 64)

	if err != nil {
		return 0, errors.BadRequestError(fmt.Sprintf("Invalid %s ID", paramName))
	}

	return id, nil
}

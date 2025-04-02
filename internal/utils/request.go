package utils

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
)

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

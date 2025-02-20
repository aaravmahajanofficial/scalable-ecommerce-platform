package handlers

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

func validateMethod(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		slog.Warn("Invalid request method",
			slog.String("method", r.Method),
			slog.String("endpoint", r.URL.Path),
		)
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return false
	}

	return true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dest interface{}) error {
	defer r.Body.Close()

	err := json.NewDecoder(r.Body).Decode(&dest)

	if errors.Is(err, io.EOF) {
		slog.Warn("Empty request body", slog.String("endpoint", r.URL.Path))
		response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("❌ Bad Request: request body cannot be empty")))
		return err
	}

	if err != nil {
		slog.Error("Failed to decode request body",
			slog.String("error", err.Error()),
			slog.String("endpoint", r.URL.Path),
		)
		response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
		return err
	}

	return nil
}

func validateStruct(w http.ResponseWriter, validate *validator.Validate, data interface{}) bool {
	if err := validate.Struct(data); err != nil {
		if validationErrs, ok := err.(validator.ValidationErrors); ok {
			slog.Warn("User input validation failed",
				slog.String("error", validationErrs.Error()),
			)
			response.WriteJson(w, http.StatusBadRequest, response.ValidationError(validationErrs))
		} else {
			slog.Error("Unexpected validation error", slog.String("error", err.Error()))
			response.WriteJson(w, http.StatusInternalServerError, response.GeneralError(err))
		}
		return false
	}
	return true
}

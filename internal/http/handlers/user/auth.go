package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/types"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate = validator.New()

var users = make(map[string]types.User)

func Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Check for correct HTTP method
		if r.Method != http.MethodPost {
			slog.Warn("Invalid request method",
				slog.String("method", r.Method),
				slog.String("endpoint", r.URL.Path),
			)
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		// Decode the request body

		var user types.User

		err := json.NewDecoder(r.Body).Decode(&user)

		if errors.Is(err, io.EOF) {
			slog.Warn("Empty request body",
				slog.String("endpoint", r.URL.Path),
			)
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("❌ Bad Request: request body cannot be empty")))
			return
		}

		if err != nil {
			slog.Error("Failed to decode request body",
				slog.String("error", err.Error()),
				slog.String("endpoint", r.URL.Path),
			)
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		if err := validate.Struct(user); err != nil {

			slog.Warn("User input validation failed",
				slog.String("endpoint", r.URL.Path),
				slog.String("error", err.Error()),
			)

			response.WriteJson(w, http.StatusBadRequest, response.ValidationError(err.(validator.ValidationErrors)))
			return

		}

		user.ID = uuid.NewString()
		users[user.ID] = user

		slog.Info("User registered successfully",
			slog.String("userId", user.ID),
			slog.String("endpoint", r.URL.Path),
		)

		response.WriteJson(w, http.StatusCreated, map[string]string{"id": user.ID})

	}
}

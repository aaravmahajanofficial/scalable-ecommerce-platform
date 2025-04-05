package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type contextKey uuid.UUID

var UserContextKey = contextKey(uuid.New())

type AuthMiddleware struct {
	jwtKey []byte
}

func NewAuthMiddleware(jwtKey []byte) *AuthMiddleware {

	return &AuthMiddleware{jwtKey: jwtKey}

}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			slog.Warn("Missing authorization header",
				slog.String("endpoint", r.URL.Path),
				slog.String("method", r.Method))
			response.Error(w, errors.UnauthorizedError("Authorization header is required"))
			return
		}

		// Token is of format : "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")

		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			slog.Warn("Invalid authorization header format",
				slog.String("header", authHeader),
				slog.String("endpoint", r.URL.Path))
			response.Error(w, errors.UnauthorizedError("Invalid authorization format"))
			return
		}

		tokenString := tokenParts[1]

		// Stores the decoded information
		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
			// check the signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {

				return nil, errors.BadRequestError("unexpected signing method")

			}
			return m.jwtKey, nil
		})

		if err != nil {
			slog.Warn("JWT parsing failed",
				slog.String("error", err.Error()),
				slog.String("endpoint", r.URL.Path))
			response.Error(w, errors.UnauthorizedError("Invalid or expired token"))
			return
		}

		if !token.Valid {
			slog.Warn("Invalid token", slog.String("endpoint", r.URL.Path))
			response.Error(w, errors.UnauthorizedError("Invalid token"))
			return
		}

		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			slog.Warn("Expired token",
				slog.String("userId", claims.UserID.String()),
				slog.String("endpoint", r.URL.Path))
			response.Error(w, errors.UnauthorizedError("Token expired"))
			return
		}

		// Add userId to the context
		// It attaches a new key-value pair ("user": claims) to the context.
		ctx := context.WithValue(r.Context(), UserContextKey, claims)

		slog.Info("User authenticated",
			slog.String("userId", claims.UserID.String()),
			slog.String("endpoint", r.URL.Path),
			slog.String("method", r.Method))

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
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

		logger := LoggerFromContext(r.Context())

		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			logger.Warn("Missing authorization header")
			response.Error(w, appErrors.UnauthorizedError("Authorization header is required"))
			return
		}

		// Token is of format : "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")

		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			logger.Warn("Invalid authorization header format", slog.String("header", authHeader))
			response.Error(w, appErrors.UnauthorizedError("Invalid authorization format"))
			return
		}

		tokenString := tokenParts[1]

		// Stores the decoded information
		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
			// check the signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok || t.Header["alg"] != jwt.SigningMethodHS256.Alg() {

				logger.Error("Unexpected signing method used in JWT", slog.Any("alg", t.Header["alg"]))
				return nil, appErrors.BadRequestError("unexpected signing method")

			}
			return m.jwtKey, nil
		})

		if err != nil {
			logger.Warn("JWT parsing failed", slog.String("error", err.Error()))
			var appErr *appErrors.AppError
			if errors.As(err, &appErr) && appErr.Code == appErrors.ErrCodeBadRequest {
				response.Error(w, appErr) // Respond with the specific bad request error
			} else {
				// Handle other parsing errors (expired, malformed, invalid signature) as Unauthorized
				response.Error(w, appErrors.UnauthorizedError("Invalid or expired token"))
			}
			return
		}

		if !token.Valid {
			logger.Warn("Invalid token")
			response.Error(w, appErrors.UnauthorizedError("Invalid token"))
			return
		}

		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			logger.Warn("Expired token", slog.String("userId", claims.UserID.String()))
			response.Error(w, appErrors.UnauthorizedError("Token expired"))
			return
		}

		// Add userId to the context
		// It attaches a new key-value pair ("user": claims) to the context.
		ctx := context.WithValue(r.Context(), UserContextKey, claims)

		requestScopedLogger := logger.With(slog.String("userId", claims.UserID.String()))
		ctx = context.WithValue(ctx, LoggerKey, requestScopedLogger)

		requestScopedLogger.Info("User authenticated")

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

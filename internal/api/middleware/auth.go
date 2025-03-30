package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/golang-jwt/jwt/v5"
)

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
			response.WriteJson(w, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		// Token is of format : "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")

		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			response.WriteJson(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		tokenString := tokenParts[1]

		// Stores the decoded information
		claims := &models.Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
			// check the signing method
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {

				return nil, errors.New("unexpected signing method")

			}
			return m.jwtKey, nil
		})

		if err != nil {
			response.WriteJson(w, http.StatusUnauthorized, "Invalid or expired token: "+err.Error())
			return
		}

		if !token.Valid {
			response.WriteJson(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			response.WriteJson(w, http.StatusUnauthorized, "Token expired")
			return
		}

		// Set the header "X-User-ID"
		r.Header.Set("X-User-ID", claims.UserID)

		// Add userId to the context
		// It attaches a new key-value pair ("user": claims) to the context.
		ctx := context.WithValue(r.Context(), "user", claims)
		next.ServeHTTP(w, r.WithContext(ctx))

	}
}

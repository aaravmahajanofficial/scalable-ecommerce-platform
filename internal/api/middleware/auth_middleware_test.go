package middleware_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testJwtKey = []byte("test-secret-key-123456789012345")

func createTestToken(userID uuid.UUID, email string, duration time.Duration, key []byte, method jwt.SigningMethod) (string, error) {
	claims := &models.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(method, claims)

	return token.SignedString(key)
}

func TestAuthMiddleware(t *testing.T) {
	// Arrange
	authMiddleware := middleware.NewAuthMiddleware(testJwtKey)
	userID := uuid.New()
	userEmail := "test@example.com"

	// Mock handler to check if the request reaches the next handler
	// and to verify the context values.
	mockNextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user claims are correctly added to the context
		claims, ok := r.Context().Value(middleware.UserContextKey).(*models.Claims)
		require.True(t, ok, "User claims should be in context")
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, userEmail, claims.Email)

		// Check if the logger with userId is in the context
		logger := middleware.LoggerFromContext(r.Context())
		require.NotNil(t, logger)

		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"success": true}`))
		require.NoError(t, err)
	})

	tests := []struct {
		name           string
		authHeader     string
		setupRequest   func(req *http.Request)
		expectedStatus int
		expectedBody   string
		expectNextCall bool
	}{
		{
			name: "Success - Valid Token",
			authHeader: func() string {
				token, err := createTestToken(userID, userEmail, time.Hour, testJwtKey, jwt.SigningMethodHS256)
				require.NoError(t, err)

				return "Bearer " + token
			}(),
			expectedStatus: http.StatusOK,
			expectedBody:   `{"success": true}`,
			expectNextCall: true,
		},
		{
			name:           "Fail - Missing Authorization Header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"success": false, "error": {"code": "UNAUTHORIZED", "message": "Authorization header is required"}}`,
			expectNextCall: false,
		},
		{
			name:           "Fail - Invalid Authorization Header Format (No Bearer)",
			authHeader:     "InvalidTokenFormat",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"success": false, "error": {"code": "UNAUTHORIZED", "message": "Invalid authorization format"}}`,
			expectNextCall: false,
		},
		{
			name:           "Fail - Invalid Authorization Header Format (Only Bearer)",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"success": false, "error": {"code": "UNAUTHORIZED", "message": "Invalid or expired token"}}`,
			expectNextCall: false,
		},
		{
			name:           "Fail - Invalid Token (Malformed)",
			authHeader:     "Bearer not.a.valid.token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"success": false, "error": {"code": "UNAUTHORIZED", "message": "Invalid or expired token"}}`, // Parsing errors
			expectNextCall: false,
		},
		{
			name: "Fail - Invalid Token (Wrong Signing Key)",
			authHeader: func() string {
				wrongKey := []byte("different-secret-key-0987654321")
				token, err := createTestToken(userID, userEmail, time.Hour, wrongKey, jwt.SigningMethodHS256)
				require.NoError(t, err)

				return "Bearer " + token
			}(),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"success": false, "error": {"code": "UNAUTHORIZED", "message": "Invalid or expired token"}}`, // Signature verification failure
			expectNextCall: false,
		},
		{
			name: "Fail - Invalid Token (Wrong Signing Method)",
			authHeader: func() string {
				token, err := createTestToken(userID, userEmail, time.Hour, testJwtKey, jwt.SigningMethodHS512)
				require.NoError(t, err)

				return "Bearer " + token
			}(),
			// This specific scenario (unexpected signing method) returns BadRequest in the code
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `{"success": false, "error": {"code": "BAD_REQUEST", "message": "unexpected signing method"}}`,
			expectNextCall: false,
		},
		{
			name: "Fail - Expired Token",
			authHeader: func() string {
				token, err := createTestToken(userID, userEmail, -time.Hour, testJwtKey, jwt.SigningMethodHS256) // Expired 1 hour ago
				require.NoError(t, err)

				return "Bearer " + token
			}(),
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"success": false, "error": {"code": "UNAUTHORIZED", "message": "Invalid or expired token"}}`,
			expectNextCall: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}

			// Add a base logger to the context, simulating the Logging middleware
			baseLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			ctx := context.WithValue(req.Context(), middleware.LoggerKey, baseLogger)
			req = req.WithContext(ctx)

			// Apply any specific request setup
			if tc.setupRequest != nil {
				tc.setupRequest(req)
			}

			rr := httptest.NewRecorder()

			// Create a handler chain: AuthMiddleware -> mockNextHandler
			handlerToTest := authMiddleware.Authenticate(mockNextHandler)

			// Act
			handlerToTest.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, tc.expectedStatus, rr.Code, "Unexpected status code")

			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, rr.Body.String(), "Unexpected response body")
			}
			// Check if mockNextHandler was called (or not called) as expected
			// This is implicitly checked by the StatusOK and "OK" body in the success case,
			// and by the error status/body in failure cases. If mockNextHandler wasn't called
			// in the success case, the status would not be 200 OK.
		})
	}
}

func TestNewAuthMiddleware(t *testing.T) {
	key := []byte("some-key")
	mw := middleware.NewAuthMiddleware(key)
	assert.NotNil(t, mw, "Middleware should not be nil")
}

package testutils

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/google/uuid"
)

func CreateTestRequestWithContext(method, target string, body io.Reader, userID uuid.UUID, pathParams map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, body)

	for key, value := range pathParams {
		req.SetPathValue(key, value)
	}

	claims := &models.Claims{UserID: userID, Email: "test@example.com"}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
	ctx = context.WithValue(ctx, middleware.LoggerKey, logger)

	return req.WithContext(ctx)
}

func CreateTestRequestWithoutContext(method, target string, body io.Reader, pathParams map[string]string) *http.Request {
	req := httptest.NewRequest(method, target, body)

	for key, value := range pathParams {
		req.SetPathValue(key, value)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx := context.WithValue(req.Context(), middleware.LoggerKey, logger)

	return req.WithContext(ctx)
}

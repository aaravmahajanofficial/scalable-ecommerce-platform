package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/handlers"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/api/middleware"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services/mocks"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils/response"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestUserHandler_Register(t *testing.T) {
	mockUserService := mocks.NewMockUserService(t)
	userHandler := handlers.NewUserHandler(mockUserService)

	t.Run("Success - User Registration", func(t *testing.T) {
		// Arrange
		registerReq := &models.RegisterRequest{
			Name:     "Test User",
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		reqBody, err := json.Marshal(registerReq)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content/Type", "application/json")

		w := httptest.NewRecorder()

		createdUser := &models.User{
			ID:    uuid.New(),
			Email: registerReq.Email,
			Name:  registerReq.Name,
		}

		// did the handler pass the right data to the service?
		mockUserService.On("Register", mock.Anything, mock.MatchedBy(func(r *models.RegisterRequest) bool {
			return r.Email == registerReq.Email && r.Name == registerReq.Name
		})).Return(createdUser, nil).Once()

		// Act
		handler := userHandler.Register()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusCreated, w.Code)

		// Put the JSON response into APIResponse, and treat the Data field as an any (i.e., interface{})

		var respBody response.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.True(t, respBody.Success)

		// why to marshall again?
		// Because, Go doesn’t know the shape of Data. So it parses it like this:
		// Objects become map[string]interface{}
		// Numbers become float64
		// map[string]interface{}{
		// 	"id": "uuid-123",
		// 	"email": "test@example.com",
		// 	"name": "Test User",
		//   }

		// We wil convert this to JSON and then to Go struct
		// resp.Data == messyData
		jsonData, err := json.Marshal(respBody.Data)
		assert.NoError(t, err)

		var extractedUserData *models.User
		err = json.Unmarshal(jsonData, &extractedUserData)
		assert.NoError(t, err)

		assert.Equal(t, createdUser.ID, extractedUserData.ID)
		assert.Equal(t, createdUser.Name, extractedUserData.Name)
		assert.Equal(t, createdUser.Email, extractedUserData.Email)

		mockUserService.AssertExpectations(t)
	})
	t.Run("Failure - Invalid Input", func(t *testing.T) {
		// Arrange
		invalidReq := struct {
			Email string `json:"email"`
		}{
			Email: "test@example.com",
		}

		reqBody, err := json.Marshal(invalidReq)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content/Type", "application/json")

		w := httptest.NewRecorder()

		// Act
		handler := userHandler.Register()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var respBody response.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.False(t, respBody.Success)
		assert.NotNil(t, respBody.Error)
		assert.Equal(t, errors.ErrCodeValidation, respBody.Error.Code)

		mockUserService.AssertNotCalled(t, "Register")
	})
	t.Run("Failure - Service Error", func(t *testing.T) {
		// Arrange
		registerReq := &models.RegisterRequest{
			Name:     "Test User",
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		reqBody, err := json.Marshal(registerReq)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", bytes.NewBuffer(reqBody))
		req.Header.Set("Content/Type", "application/json")

		w := httptest.NewRecorder()

		// did the handler pass the right data to the service?
		mockUserService.On("Register", mock.Anything, mock.MatchedBy(func(r *models.RegisterRequest) bool {
			return r.Email == registerReq.Email && r.Name == registerReq.Name
		})).Return(nil, errors.DuplicateEntryError("Email already registered")).Once()

		// Act
		handler := userHandler.Register()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusConflict, w.Code)

		var respBody response.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.False(t, respBody.Success)
		assert.NotNil(t, respBody.Error)
		assert.Equal(t, errors.ErrCodeDuplicateEntry, respBody.Error.Code)

		mockUserService.AssertExpectations(t)
	})
}

func TestUserHandler_Login(t *testing.T) {
	mockUserService := mocks.NewMockUserService(t)
	userHandler := handlers.NewUserHandler(mockUserService)

	t.Run("Success - Valid Login", func(t *testing.T) {
		// Arrange
		loginReq := &models.LoginRequest{
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		reqBody, err := json.Marshal(loginReq)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content/Type", "application/json")

		w := httptest.NewRecorder()

		loginResp := &models.LoginResponse{
			Success:   true,
			Token:     "jwt-token",
			ExpiresIn: 86400,
		}

		// did the handler pass the right data to the service?
		mockUserService.On("Login", mock.Anything, mock.MatchedBy(func(r *models.LoginRequest) bool {
			return r.Email == loginReq.Email && r.Password == loginReq.Password
		})).Return(loginResp, nil).Once()

		// Act
		handler := userHandler.Login()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var respBody response.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.True(t, respBody.Success)

		// Extract the login resp
		responseData, err := json.Marshal(respBody.Data)
		assert.NoError(t, err)

		var returnedResp models.LoginResponse
		err = json.Unmarshal(responseData, &returnedResp)
		assert.NoError(t, err)

		assert.True(t, returnedResp.Success)
		assert.Equal(t, loginResp.Token, returnedResp.Token)
		assert.Equal(t, loginResp.ExpiresIn, returnedResp.ExpiresIn)

		mockUserService.AssertExpectations(t)
	})
	t.Run("Failure - Invalid Credentials", func(t *testing.T) {
		// Arrange
		loginReq := &models.LoginRequest{
			Email:    "test@example.com",
			Password: "WrongP@ssword123!",
		}

		reqBody, err := json.Marshal(loginReq)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content/Type", "application/json")

		w := httptest.NewRecorder()

		loginResp := &models.LoginResponse{
			Success:        false,
			Message:        "Invalid Email or Password",
			RemainingTries: 4,
		}

		// did the handler pass the right data to the service?
		mockUserService.On("Login", mock.Anything, mock.MatchedBy(func(r *models.LoginRequest) bool {
			return r.Email == loginReq.Email && r.Password == loginReq.Password
		})).Return(loginResp, nil).Once()

		// Act
		handler := userHandler.Login()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var respBody response.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.False(t, respBody.Success)
		assert.NotNil(t, respBody.Error)
		assert.Equal(t, errors.ErrCodeUnauthorized, respBody.Error.Code)

		mockUserService.AssertExpectations(t)
	})
	t.Run("Failure - Rate Limited", func(t *testing.T) {
		// Arrange
		loginReq := &models.RegisterRequest{
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		reqBody, err := json.Marshal(loginReq)
		assert.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", bytes.NewBuffer(reqBody))
		req.Header.Set("Content/Type", "application/json")

		w := httptest.NewRecorder()

		loginResp := &models.LoginResponse{
			Success:    false,
			Message:    "Too many login attempts. Please try again later.",
			RetryAfter: 30,
		}

		// did the handler pass the right data to the service?
		mockUserService.On("Login", mock.Anything, mock.MatchedBy(func(r *models.LoginRequest) bool {
			return r.Email == loginReq.Email && r.Password == loginReq.Password
		})).Return(loginResp, nil).Once()

		// Act
		handler := userHandler.Login()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusTooManyRequests, w.Code)

		var respBody response.APIResponse
		err = json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.False(t, respBody.Success)
		assert.NotNil(t, respBody.Error)
		assert.Equal(t, errors.ErrCodeTooManyRequests, respBody.Error.Code)

		mockUserService.AssertExpectations(t)
	})
}

func TestUserHandler_Profile(t *testing.T) {
	mockUserService := mocks.NewMockUserService(t)
	userHandler := handlers.NewUserHandler(mockUserService)

	t.Run("Success - Get Profile", func(t *testing.T) {
		// Arrange
		user := &models.User{
			ID:    uuid.New(),
			Email: "test@example.com",
			Name:  "Test User",
		}

		mockUserService.On("GetUserByID", mock.Anything, user.ID).Return(user, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)

		claims := &models.Claims{
			UserID: user.ID,
			Email:  user.Email,
		}

		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		// Act
		handler := userHandler.Profile()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusOK, w.Code)

		var respBody response.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.True(t, respBody.Success)

		mockUserService.AssertExpectations(t)
	})
	t.Run("Failure - No Auth Context", func(t *testing.T) {
		// Arrange - request without auth context
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
		w := httptest.NewRecorder()

		// Act
		handler := userHandler.Profile()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var respBody response.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.False(t, respBody.Success)
		assert.NotNil(t, respBody.Error)
		assert.Equal(t, errors.ErrCodeUnauthorized, respBody.Error.Code)

		mockUserService.AssertNotCalled(t, "GetUserByID")
	})
	t.Run("Failure - User Not Found", func(t *testing.T) {
		// Arrange
		userID := uuid.New()
		email := "test@example.com"

		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)

		claims := &models.Claims{
			UserID: userID,
			Email:  email,
		}

		ctx := context.WithValue(req.Context(), middleware.UserContextKey, claims)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()

		mockUserService.On("GetUserByID", mock.Anything, userID).Return(nil, errors.NotFoundError("User not found")).Once()

		// Act
		handler := userHandler.Profile()
		handler(w, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, w.Code)

		var respBody response.APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &respBody)
		assert.NoError(t, err)
		assert.False(t, respBody.Success)
		assert.NotNil(t, respBody.Error)
		assert.Equal(t, errors.ErrCodeNotFound, respBody.Error.Code)

		mockUserService.AssertExpectations(t)
	})
}

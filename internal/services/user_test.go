package service_test

import (
	"errors"
	"testing"
	"time"

	appErrors "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/errors"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/mocks"
	service "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/services"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestUserService_Register(t *testing.T) {
	// Arrange
	mockUserRepo := mocks.NewMockUserRepository(t)
	mockRedisRepo := mocks.NewMockRateLimitRepository(t)
	jwtKey := []byte("test-key")

	userService := service.NewUserService(mockUserRepo, mockRedisRepo, jwtKey)

	t.Run("Success - User Registration", func(t *testing.T) {
		ctx := t.Context()
		req := &models.RegisterRequest{
			Name:     "Test User",
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		// Mock Behavior -> email is fresh!
		mockUserRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(nil, nil).Once()

		// Mock Behavior -> user was created
		// mock.AnythingOfType is used when, you don't know the exact value of the user struct, as here, password field may contain hashedPassword
		mockUserRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil).Once()

		// Act
		user, err := userService.Register(ctx, req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, req.Name, user.Name)
		assert.Equal(t, req.Email, user.Email)

		// Verify that password was hashed by bcrypt
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
		assert.NoError(t, err)

		mockUserRepo.AssertExpectations(t)
	})
	t.Run("Failure - Duplicate Email", func(t *testing.T) {
		ctx := t.Context()
		req := &models.RegisterRequest{
			Name:     "Test User",
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		exisitingUser := &models.User{
			ID:    uuid.New(),
			Email: req.Email,
		}

		// Mock Behavior -> email is not fresh!
		mockUserRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(exisitingUser, nil).Once()

		// Act
		user, err := userService.Register(ctx, req)

		// Assert
		assert.Nil(t, user)
		assert.Error(t, err)

		// Check if the error is of type appError
		var appErr *appErrors.AppError

		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDuplicateEntry, appErr.Code)

		mockUserRepo.AssertExpectations(t)
	})

	t.Run("Failure - Database Error", func(t *testing.T) {
		ctx := t.Context()
		req := &models.RegisterRequest{
			Name:     "Test User",
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		// Mock Behavior -> email is fresh!
		mockUserRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(nil, nil).Once()

		// Mock Behavior -> something is wrong with database
		dbErr := errors.New("something exploaded")
		mockUserRepo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(dbErr).Once()

		// Act
		user, err := userService.Register(ctx, req)

		// Assert
		assert.Nil(t, user)
		assert.Error(t, err)

		// Check if the error is of type appError
		var appErr *appErrors.AppError

		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeDatabaseError, appErr.Code)

		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserService_Login(t *testing.T) {
	mockUserRepo := mocks.NewMockUserRepository(t)
	mockRedisRepo := mocks.NewMockRateLimitRepository(t)
	jwtKey := []byte("test-key")

	userService := service.NewUserService(mockUserRepo, mockRedisRepo, jwtKey)

	t.Run("Success - Valid Credentials", func(t *testing.T) {
		// Arrange
		ctx := t.Context()
		password := "P@ssword123!"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		req := &models.LoginRequest{
			Email:    "test@example.com",
			Password: password,
		}

		user := &models.User{
			ID:       uuid.New(),
			Email:    req.Email,
			Password: string(hashedPassword),
			Name:     "Test User",
		}

		// Mock Behavior -> rate limit check
		mockRedisRepo.On("CheckLoginRateLimit", mock.Anything, req.Email).Return(true, 5, 0, nil).Once()

		// Mock Behavior -> user exists!
		mockUserRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(user, nil).Once()

		// Act
		resp, err := userService.Login(ctx, req)

		// Assert
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
		assert.NotEmpty(t, resp.Token)

		// Verify if JWT returned by service is:
		// ✅ properly signed
		// ✅ has valid claims (like email, expiry)
		// ✅ can be parsed without errors

		// actual token, where the token be decoded, server/secret-key
		token, err := jwt.ParseWithClaims(resp.Token, &models.Claims{}, func(_ *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		assert.NoError(t, err)

		claims, ok := token.Claims.(*models.Claims)
		assert.True(t, ok)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)

		mockUserRepo.AssertExpectations(t)
		mockRedisRepo.AssertExpectations(t)
	})
	t.Run("Failure - Invalid Password", func(t *testing.T) {
		// Arrange
		ctx := t.Context()
		password := "P@ssword123!"
		wrongPassword := "WrongP@ssword123!"
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		require.NoError(t, err)

		req := &models.LoginRequest{
			Email:    "test@example.com",
			Password: wrongPassword,
		}

		user := &models.User{
			ID:       uuid.New(),
			Email:    req.Email,
			Password: string(hashedPassword),
			Name:     "Test User",
		}

		// Mock Behavior -> within limits
		mockRedisRepo.On("CheckLoginRateLimit", mock.Anything, req.Email).Return(true, 4, 0, nil).Once()

		// Mock Behavior -> user exists, we can't return any error, otherwise we would miss the password check
		mockUserRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(user, nil).Once()

		// Act
		resp, err := userService.Login(ctx, req)

		// Assert
		assert.NoError(t, err) // no system level failure
		assert.NotNil(t, resp) // no system level failure
		assert.False(t, resp.Success)
		assert.Equal(t, 4, resp.RemainingTries)
		assert.Empty(t, resp.Token)

		mockUserRepo.AssertExpectations(t)
		mockRedisRepo.AssertExpectations(t)
	})
	t.Run("Failure - Rate Limited", func(t *testing.T) {
		// Arrange
		ctx := t.Context()
		req := &models.LoginRequest{
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		// Mock Behavior -> within limits
		mockRedisRepo.On("CheckLoginRateLimit", mock.Anything, req.Email).Return(false, 0, 30, nil).Once()

		// Act
		resp, err := userService.Login(ctx, req)

		// Assert
		assert.NoError(t, err) // no system level failure
		assert.NotNil(t, resp) // no system level failure
		assert.False(t, resp.Success)
		assert.Equal(t, 30, resp.RetryAfter)
		assert.Empty(t, resp.Token)

		mockRedisRepo.AssertExpectations(t)
		// Check "GetUserByEmail" is not called
		mockUserRepo.AssertNotCalled(t, "GetUserByEmail")
	})

	t.Run("Failure - User Not found", func(t *testing.T) {
		// Arrange
		ctx := t.Context()

		req := &models.LoginRequest{
			Email:    "test@example.com",
			Password: "P@ssword123!",
		}

		// Mock Behavior -> within limits
		mockRedisRepo.On("CheckLoginRateLimit", mock.Anything, req.Email).Return(true, 5, 0, nil).Once()

		// Mock Behavior -> user not fresh!
		mockUserRepo.On("GetUserByEmail", mock.Anything, req.Email).Return(nil, errors.New("User not found")).Once()

		// Act
		resp, err := userService.Login(ctx, req)

		// Assert
		assert.NoError(t, err) // no system level failure
		assert.NotNil(t, resp) // no system level failure
		assert.False(t, resp.Success)
		assert.Equal(t, 5, resp.RemainingTries)
		assert.Empty(t, resp.Token)

		mockUserRepo.AssertExpectations(t)
		mockRedisRepo.AssertExpectations(t)
	})
}

func TestUserService_GetUserByID(t *testing.T) {
	mockUserRepo := mocks.NewMockUserRepository(t)
	mockRedisRepo := mocks.NewMockRateLimitRepository(t)
	jwtKey := []byte("test-key")

	userService := service.NewUserService(mockUserRepo, mockRedisRepo, jwtKey)

	t.Run("Success - User Found", func(t *testing.T) {
		// Arrange
		ctx := t.Context()
		userID := uuid.New()

		exisitingUser := &models.User{
			ID:        userID,
			Email:     "test@example.com",
			Name:      "Test User",
			CreatedAt: time.Now().Add(-24 * time.Hour),
			UpdatedAt: time.Now(),
		}

		// Mock Behavior -> user not fresh!
		mockUserRepo.On("GetUserByID", mock.Anything, userID).Return(exisitingUser, nil).Once()

		// Act
		resp, err := userService.GetUserByID(ctx, userID)

		// Assert
		assert.NoError(t, err) // no system level failure
		assert.NotNil(t, resp) // no system level failure
		assert.Equal(t, exisitingUser.ID, resp.ID)
		assert.Equal(t, exisitingUser.Email, resp.Email)
		assert.Equal(t, exisitingUser.Name, resp.Name)

		mockUserRepo.AssertExpectations(t)
	})
	t.Run("Failure - User not Found", func(t *testing.T) {
		// Arrange
		ctx := t.Context()
		userID := uuid.New()

		// Mock Behavior -> user not fresh!
		mockUserRepo.On("GetUserByID", mock.Anything, userID).Return(nil, errors.New("User not found")).Once()

		// Act
		resp, err := userService.GetUserByID(ctx, userID)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, resp)

		// Check if the error is of type appError
		var appErr *appErrors.AppError

		assert.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.ErrCodeNotFound, appErr.Code)

		mockUserRepo.AssertExpectations(t)
	})
}

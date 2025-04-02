package service

import (
	"context"
	"errors"
	"time"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	repository "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repositories/redis"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo      *repository.UserRepository
	redisRepo *redis.RedisRepo
	jwtKey    []byte
}

func NewUserService(repo *repository.UserRepository, redisRepo *redis.RedisRepo, jwtKey []byte) *UserService {
	return &UserService{
		repo:      repo,
		redisRepo: redisRepo,
		jwtKey:    jwtKey,
	}
}

func (s *UserService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	err = s.repo.CreateUser(ctx, user)

	if err != nil {
		return nil, err
	}

	return user, err

}

func (s *UserService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {

	// check rate limit
	allowed, remaining, retryAfter, err := s.redisRepo.CheckLoginRateLimit(ctx, req.Email)

	if err != nil {
		return nil, err
	}

	if !allowed {
		return &models.LoginResponse{
			Success:    false,
			Message:    "Too many login attempts. Please try again later.",
			RetryAfter: retryAfter,
		}, nil
	}

	// Retrieve the user from the DB and compare the passwords
	user, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return &models.LoginResponse{
			Success:        false,
			Message:        "Invalid email or password",
			RemainingTries: remaining,
		}, nil
	}

	claims := &models.Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Generate Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		return nil, errors.New("failed to generate authentication token")
	}

	return &models.LoginResponse{
		Success:   true,
		Token:     tokenString,
		ExpiresIn: int(time.Until(claims.ExpiresAt.Time).Seconds()),
	}, nil

}

func (s *UserService) GetUserByID(ctx context.Context, id string) (*models.User, error) {

	user, err := s.repo.GetUserById(ctx, id)

	if err != nil {
		return nil, err
	}

	// Note: Password is already included in repository query
	return user, nil

}

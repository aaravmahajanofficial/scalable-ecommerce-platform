package service

import (
	"errors"
	"time"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo   *repository.UserRepository
	jwtKey []byte
}

func NewUserService(repo *repository.UserRepository, jwtKey []byte) *UserService {
	return &UserService{
		repo:   repo,
		jwtKey: jwtKey,
	}
}

func (s *UserService) Register(req *models.RegisterRequest) (*models.User, error) {

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

	err = s.repo.CreateUser(user)

	if err != nil {
		return nil, err
	}

	return user, err

}

func (s *UserService) Login(req *models.LoginRequest) (string, error) {

	// Retrieve the user from the DB
	user, err := s.repo.GetUserByEmail(req.Email)

	if err != nil {
		return "", err
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))

	if err != nil {
		return "", errors.New("invalid credentials")
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
		return "", nil
	}

	return tokenString, nil

}

func (s *UserService) GetUserByID(id string) (*models.User, error) {

	user, err := s.repo.GetUserById(id)

	if err != nil {
		return nil, err
	}

	// Note: Password is already included in repository query
	return user, nil

}

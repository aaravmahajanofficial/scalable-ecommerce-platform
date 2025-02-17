package storage

import (
	model "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
)

type Storage interface {
	CreateUser(user *model.User) error
	GetUserByEmail(email string) (*model.User, error)
}

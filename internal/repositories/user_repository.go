package repository

import (
	"context"
	"database/sql"
	"errors"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/utils"
	"github.com/google/uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserById(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type userRepository struct {
	DB *sql.DB
}

func NewUserRepo(db *sql.DB) UserRepository {
	return &userRepository{DB: db}
}

func (r *userRepository) CreateUser(ctx context.Context, user *models.User) error {

	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	query := `
		INSERT INTO users(email, password, name, created_at, updated_at)
		VALUES($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return r.DB.QueryRowContext(dbCtx, query, user.Email, user.Password, user.Name).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

}

func (r *userRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {

	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	user := &models.User{} // user holds the address of the new instance of new User models
	query := `SELECT id, email, password, name, created_at, updated_at
			  FROM users 
			  WHERE email = $1`

	err := r.DB.QueryRowContext(dbCtx, query, email).Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil

}

func (r *userRepository) GetUserById(ctx context.Context, id uuid.UUID) (*models.User, error) {

	dbCtx, cancel := utils.WithDBTimeout(ctx)
	defer cancel()

	user := &models.User{}

	query := `
	SELECT id, email, name, created_at, updated_at
	FROM users
	WHERE id = $1
	`

	err := r.DB.QueryRowContext(dbCtx, query, id).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {

		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}

		return nil, err

	}

	return user, nil

}

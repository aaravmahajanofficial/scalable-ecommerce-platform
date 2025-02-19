package repository

import (
	"database/sql"
	"errors"

	models "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

func (p *UserRepository) CreateUser(user *models.User) error {

	query := `
		INSERT INTO users(email, password, name, created_at, updated_at)
		VALUES($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return p.DB.QueryRow(query, user.Email, user.Password, user.Name).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

}

func (p *UserRepository) GetUserByEmail(email string) (*models.User, error) {

	user := &models.User{} // user holds the address of the new instance of new User models
	query := `SELECT id, email, password, name, created_at, updated_at
			  FROM users 
			  WHERE email = $1`

	err := p.DB.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil

}

func (p *UserRepository) GetUserById(id string) (*models.User, error) {

	user := &models.User{}

	query := `
	SELECT id, email, name, created_at, updated_at
	FROM users
	WHERE id = $1
	`

	err := p.DB.QueryRow(query, id).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {

		if err == sql.ErrNoRows {
			return nil, errors.New("user not found")
		}

		return nil, err

	}

	return user, nil

}

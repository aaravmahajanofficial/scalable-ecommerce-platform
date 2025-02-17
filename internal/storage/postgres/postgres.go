package postgres

import (
	"database/sql"

	"github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/config"
	model "github.com/aaravmahajanofficial/scalable-ecommerce-platform/internal/models"
	_ "github.com/lib/pq"
)

type Postgres struct {
	db *sql.DB
}

func New(cfg *config.Config) (*Postgres, error) {

	db, err := sql.Open("postgres", cfg.StoragePath)

	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users(
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		name VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW() NOT NULL,
		updated_at TIMESTAMP DEFAULT NOW() NOT NULL
	)`)

	// Test the connection to make sure DB is reachable
	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Postgres{db: db}, nil

}

func (p *Postgres) Close() error {
	return p.db.Close()
}

func (p *Postgres) CreateUser(user *model.User) error {

	query := `
		INSERT INTO users(email, password, name, created_at, updated_at)
		VALUES($1, $2, $3, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	return p.db.QueryRow(query, user.Email, user.Password, user.Name).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

}

func (p *Postgres) GetUserByEmail(email string) (*model.User, error) {

	user := &model.User{} // user holds the address of the new instance of new User model
	query := `SELECT id, email, password, name, created_at, updated_at
			  FROM users 
			  WHERE email = $1`

	err := p.db.QueryRow(query, email).Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return user, nil

}

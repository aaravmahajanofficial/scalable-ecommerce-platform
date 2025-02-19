package repository

import "database/sql"

type ProductRepository struct {
	DB *sql.DB
}

func NewProductRepo(db *sql.DB) *ProductRepository {
	return &ProductRepository{DB: db}
}

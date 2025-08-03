package data

import (
	"context"
	"database/sql"
	"time"
)

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	CategoryID  int       `json:"category_id"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"-"`
}

type ProductModel struct {
	DB *sql.DB
}

func (p *ProductModel) Insert(ctx context.Context, product *Product) error {
	return nil
}

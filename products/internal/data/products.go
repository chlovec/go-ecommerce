package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
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

type ProductRepository interface {
	Insert(ctx context.Context, product *Product) error
}

func (p *ProductModel) Insert(ctx context.Context, product *Product) error {
	query := `
		INSERT INTO products (name, category_id, description, price, quantity)
		VALUES($1, $2, $3, $4, $5)
		RETURNING id, created_at, version
	`
	args := []any{
		product.Name,
		product.CategoryID,
		product.Description,
		product.Price,
		product.Quantity,
	}
	err := p.DB.QueryRowContext(ctx, query, args...).Scan(
		&product.ID,
		&product.CreatedAt,
		&product.Version,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23503" {
			return fmt.Errorf(
				"category_id %d does not exist: %w",
				product.CategoryID,
				ErrInvalidCategoryId,
			)
		}
		return err
	}

	return nil
}

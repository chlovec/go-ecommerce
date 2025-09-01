package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
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
	query, args, _ := sq.Insert("products").
		Columns("name", "category_id", "description", "price", "quantity").
		Values(
			product.Name,
			product.CategoryID,
			product.Description,
			product.Price,
			product.Quantity).
		Suffix("RETURNING id, created_at, version").
		ToSql()
	err := p.DB.QueryRowContext(ctx, query, args...).Scan(
		&product.ID,
		&product.CreatedAt,
		&product.Version,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == ErrForeignKeyViolation {
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

func (p *ProductModel) GetByID(ctx context.Context, id int64) (*Product, error) {
	query, _, _ := sq.Select(
		"id",
		"name",
		"category_id",
		"description",
		"price",
		"quantity",
		"created_at",
		"version",
	).
		From("products").
		Where(sq.Eq{"id": id}).
		ToSql()

	var product Product
	err := p.DB.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.CategoryID,
		&product.Description,
		&product.Price,
		&product.Quantity,
		&product.CreatedAt,
		&product.Version,
	)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return &product, nil
}

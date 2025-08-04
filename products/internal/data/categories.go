package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"-"`
}

type CategoryModel struct {
	DB *sql.DB
}

type CategoryRepository interface {
	Insert(ctx context.Context, category *Category) error
	GetByID(ctx context.Context, id int64) (*Category, error)
}

func (p *CategoryModel) Insert(ctx context.Context, category *Category) error {
	query := `
		INSERT INTO categories(name, description)
		VALUES($1, $2)
		RETURNING id, created_at, version
	`
	args := []any{category.Name, category.Description}
	return p.DB.QueryRowContext(ctx, query, args...).Scan(
		&category.ID,
		&category.CreatedAt,
		&category.Version,
	)
}

func (p *CategoryModel) GetByID(ctx context.Context, id int64) (*Category, error) {
	query := `
		SELECT id, name, description, created_at, version
		FROM categories
		WHERE id = $1
	`
	var category Category
	err := p.DB.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.CreatedAt,
		&category.Version,
	)

	// Handle any errors. If there was no record found, Scan()
	// will return a sql.ErrNoRows error. Check for this and
	// return the custom ErrRecordNotFound
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return &category, nil
}

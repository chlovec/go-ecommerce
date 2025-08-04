package data

import (
	"context"
	"database/sql"
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
}

func (p *CategoryModel) Insert(ctx context.Context, category *Category) error {
	query := `
		INSERT INTO categories(name, description)
		VALUES($1, $2)
		RETURNING id, created_at, version
	`
	args := []any{category.Name, category.Description}
	return p.DB.QueryRowContext(ctx, query, args...).
		Scan(&category.ID, &category.CreatedAt, &category.Version)
}

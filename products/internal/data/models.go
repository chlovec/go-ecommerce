package data

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type ProductRepository interface {
	Insert(ctx context.Context, product *Product) error
}

type Models struct {
	Product ProductRepository
}

func NewModels(db *sql.DB) Models {
	return Models{
		Product: &ProductModel{DB: db},
	}
}

package data

import (
	"context"
	"errors"
)

var (
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
	ErrInvalidCategoryId = errors.New("invalid category_id")
)

type ProductRepository interface {
	Insert(ctx context.Context, product *Product) error
}

type Models struct {
	Product ProductRepository
}

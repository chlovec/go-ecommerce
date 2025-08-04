package data

import (
	"errors"
)

var (
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
	ErrInvalidCategoryId = errors.New("invalid category_id")
)

type Models struct {
	Product  ProductRepository
	Category CategoryRepository
}

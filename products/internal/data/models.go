package data

import (
	"errors"
)

const ErrForeignKeyViolation = "23503"

var (
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
	ErrInvalidCategoryId = errors.New("invalid category_id")
)

type Models struct {
	Product  ProductRepository
	Category CategoryRepository
}

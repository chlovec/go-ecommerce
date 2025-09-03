package handlers

import (
	"database/sql"
	"log/slog"

	"github.com/chlovec/go-ecommerce/products/internal/data"
	"github.com/go-playground/validator/v10"
)

type Handlers struct {
	logger    *slog.Logger
	validator *validator.Validate
	models    data.Models
}

func NewHandlers(logger *slog.Logger, db *sql.DB) *Handlers {
	return &Handlers{
		logger:    logger,
		validator: validator.New(),
		models: data.Models{
			Product:  data.NewProductModel(db),
			Category: data.NewCategoryModel(db),
		},
	}
}

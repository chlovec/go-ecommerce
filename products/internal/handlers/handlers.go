package handlers

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
)

type Handlers struct {
	logger    *slog.Logger
	validator *validator.Validate
}

func NewHandlers(logger *slog.Logger) *Handlers {
	return &Handlers{
		logger:    logger,
		validator: validator.New(),
	}
}

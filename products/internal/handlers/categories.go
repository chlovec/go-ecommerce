package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/chlovec/go-ecommerce/products/internal/data"
)

type categoryDTO struct {
	Name        string `json:"name"        validate:"required,min=3,max=100"`
	Description string `json:"description" validate:"omitempty"`
}

// POST v1/api/categories
func (h *Handlers) CreateCategoryHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body. If it fails, respond with 400 Bad Request. Include a user
	// friendly message with the response.
	var payload categoryDTO

	err := h.readJSON(w, r, &payload)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}

	// Validate the request body. If validation fails, respond with 422 Unprocessable
	// Entity. Include a user friendly message with the response.
	err = h.validator.Struct(payload)
	if err != nil {
		h.failedValidationResponse(w, r, err)
		return
	}

	// Save category to db. If category id does not exist, send 400 Bad Request to the
	// client. For any other error, respond send 500 Internal Server Error.
	category := data.Category{
		Name:        payload.Name,
		Description: payload.Description,
	}

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.models.Category.Insert(ctx, &category)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}

	// Write successful response if all succeeds.
	h.writeJSON(w, r, http.StatusCreated, envelope{"category": category}, nil)
}

// POST v1/api/categories
func (h *Handlers) GetCategoryByID(w http.ResponseWriter, r *http.Request) {
	// Read and validate id param.
	id, err := h.readIDParam(r)
	if err != nil || id < 1 {
		if id < 1 && err == nil {
			err = fmt.Errorf("%w: %d", ErrInvalidIDParam, id)
		}
		h.badRequestResponse(w, r, err)
		return
	}

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	category, err := h.models.Category.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			h.notFoundResponse(w, r, err)
		} else {
			h.serverErrorResponse(w, r, err)
		}
		return
	}

	env := envelope{"category": category}
	h.writeJSON(w, r, http.StatusOK, env, nil)
}

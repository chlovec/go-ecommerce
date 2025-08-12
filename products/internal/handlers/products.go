package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/chlovec/go-ecommerce/products/internal/data"
)

type productDTO struct {
	Name        string  `json:"name"        validate:"required,min=3,max=100"`
	CategoryID  int     `json:"category_id" validate:"required"`
	Description string  `json:"description" validate:"omitempty"`
	Price       float64 `json:"price"       validate:"omitempty,gte=0"`
	Quantity    int     `json:"quantity"    validate:"omitempty,gte=0"`
}

// POST v1/api/products
func (h *Handlers) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	// Parse request body. If it fails, respond with 400 Bad Request. Include a user
	// friendly message with the response.
	var payload productDTO

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

	// Save product to db. If category id does not exist, send 400 Bad Request to the
	// client. For any other error, respond send 500 Internal Server Error.
	product := data.Product{
		Name:        payload.Name,
		CategoryID:  payload.CategoryID,
		Description: payload.Description,
		Price:       payload.Price,
		Quantity:    payload.Quantity,
	}

	// Create a context with a 5-second timeout deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.models.Product.Insert(ctx, &product)
	if err != nil {
		if errors.Is(err, data.ErrInvalidCategoryId) {
			h.badRequestResponse(w, r, err)
		} else {
			h.serverErrorResponse(w, r, err)
		}

		return
	}

	// Write successful response if all succeeds.
	// Create a location header to be included in the http response to
	// let the client know which url they can find newly created
	// resource at.
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/api/products/%d", product.ID))
	h.writeJSON(w, r, http.StatusCreated, envelope{"product": product}, headers)
}

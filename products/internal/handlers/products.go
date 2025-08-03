package handlers

import (
	"net/http"

	"github.com/chlovec/go-ecommerce/products/internal/data"
)

type productDTO struct {
	Name        string  `json:"name"        validate:"required,min=3,max=100"`
	CategoryID  int     `json:"category_id" validate:"required"`
	Description string  `json:"description" validate:"omitempty"`
	Price       float64 `json:"price"       validate:"omitempty,gte=0"`
	Quantity    int     `json:"quantity"    validate:"omitempty,gte=0"`
}

var productFieldMap = map[string]string{
	"Name":        "name",
	"CategoryID":  "category_id",
	"Description": "description",
	"Price":       "price",
	"Quantity":    "quantity",
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

	// Save to db. If the db query fails and is because it couldn't find a matching
	// category respond with 400 Bad Request. If it fails because of any other issue,
	// respond with 500 Internal Server Error.
	product := data.Product{
		Name:        payload.Name,
		CategoryID:  payload.CategoryID,
		Description: payload.Description,
		Price:       payload.Price,
		Quantity:    payload.Quantity,
	}

	// Write successful response if all succeeds.
	_ = h.writeJSON(w, http.StatusCreated, envelope{"product": product}, nil)
}

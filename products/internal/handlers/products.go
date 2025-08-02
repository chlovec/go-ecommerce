package handlers

import (
	"encoding/json"
	"net/http"
)

type productDTO struct {
	Name        string  `json:"name"`
	CategoryID  int     `json:"category_id"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
}

// POST v1/api/products
func (h *Handlers) CreateProductHandler(w http.ResponseWriter, r *http.Request) {
	var product productDTO
	err := h.readJSON(w, r, &product)
	if err != nil {
		// Respond with 400 Bad Request
		h.badRequestResponse(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(product)
}

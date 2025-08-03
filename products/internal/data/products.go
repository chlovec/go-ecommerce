package data

import (
	"time"
)

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	CategoryID  int       `json:"category_id"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Version     int       `json:"version"`
	CreatedAT   time.Time `json:"-"`
}

type ProductModel interface {
}

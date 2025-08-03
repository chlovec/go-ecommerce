package handlers

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"

	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateProductHandler(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	t.Run("create product successfully", func(t *testing.T) {
		productInput := productDTO{
			Name:        "Bluetooth Headphones",
			CategoryID:  1,
			Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
			Price:       129.99,
			Quantity:    25,
		}

		body, _ := json.Marshal(productInput)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
		rw := httptest.NewRecorder()

		h := NewHandlers(logger)
		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		expectedResponse := `{
			"product": {
				"id": 0,
				"name": "Bluetooth Headphones",
				"category_id": 1,
				"description": "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
				"price": 129.99,
				"quantity": 25,
				"version": 0
			}
		}`
		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
		assert.JSONEq(
			t,
			expectedResponse,
			rw.Body.String(),
			"\nexpected %v \ngot %v",
			expectedResponse,
			rw.Body.String(),
		)

		// var actual productResponse
		// err := json.NewDecoder(res.Body).Decode(&actual)
		// assert.NoError(t, err)

		// expectedProduct := data.Product{
		// 	ID:         0,
		// 	Name:       "Bluetooth Headphones",
		// 	CategoryID: 1,
		// 	Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
		// 	Price:      129.99,
		// 	Quantity:   25,
		// }

		// assert.Equal(t, expectedProduct, actual.Product, "\nexpected %v \ngot %v", expectedProduct, actual.Product)
	})

	t.Run("request body contains unknown field", func(t *testing.T) {
		var productInput = struct {
			ID          int     `json:"id"`
			Name        string  `json:"name"`
			Description string  `json:"description"`
			Price       float64 `json:"price"`
			Quantity    int     `json:"quantity"`
		}{
			ID:          1,
			Name:        "Bluetooth Headphones",
			Description: "Noise-cancelling wireless over-ear headphones with 30-hour battery life.",
			Price:       129.99,
			Quantity:    25,
		}

		body, _ := json.Marshal(productInput)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
		rw := httptest.NewRecorder()

		h := NewHandlers(logger)
		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		expectError := envelope{"error": `json: unknown field "id"`}

		var actualError envelope
		err := json.NewDecoder(res.Body).Decode(&actualError)
		assert.NoError(t, err)

		assert.Equal(t, expectError, actualError)
	},
	)

	t.Run("request body is empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(""))
		rw := httptest.NewRecorder()

		h := NewHandlers(logger)
		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		expectError := envelope{"error": `body must not be empty`}

		var actualError envelope
		err := json.NewDecoder(res.Body).Decode(&actualError)
		assert.NoError(t, err)

		assert.Equal(t, expectError, actualError)
	})

	t.Run("failed validation", func(t *testing.T) {
		productInput := productDTO{
			Name:     "Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones",
			Price:    -1,
			Quantity: -20,
		}

		body, _ := json.Marshal(productInput)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
		rw := httptest.NewRecorder()

		h := NewHandlers(logger)
		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		expected := envelope{
			"error": map[string]interface{}{
				"category_id": "is required",
				"name":        "must be at most 100 characters long",
				"price":       "must be greater than or equal to 0",
				"quantity":    "must be greater than or equal to 0",
			},
		}

		var actual envelope
		err := json.NewDecoder(res.Body).Decode(&actual)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}

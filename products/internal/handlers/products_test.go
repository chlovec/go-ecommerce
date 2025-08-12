package handlers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"testing"

	"github.com/chlovec/go-ecommerce/products/internal/data"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) Insert(ctx context.Context, product *data.Product) error {
	args := m.Called(ctx, product)
	return args.Error(0)
}

func setupProductHandlerTest(
	t *testing.T,
	w io.Writer,
	body io.Reader,
) (*httptest.ResponseRecorder, *http.Request, Handlers, *MockProductRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(w, nil))
	req := httptest.NewRequest(http.MethodPost, "/products", body)
	rw := httptest.NewRecorder()
	mockProductRepo := new(MockProductRepository)

	handlers := Handlers{
		logger:    logger,
		validator: validator.New(),
		models: data.Models{
			Product: mockProductRepo,
		},
	}

	return rw, req, handlers, mockProductRepo
}

func TestCreateProductHandler(t *testing.T) {
	var buf bytes.Buffer
	payload := `{
		"name": "Test Product",
		"category_id": 1,
		"description": "A test product",
		"price": 19.99,
		"quantity": 10
	}`

	productToInsert := data.Product{
		Name:        "Test Product",
		CategoryID:  1,
		Description: "A test product",
		Price:       19.99,
		Quantity:    10,
	}

	t.Run("create product successfully", func(t *testing.T) {
		rw, req, h, mockProductRepo := setupProductHandlerTest(t, &buf, strings.NewReader(payload))
		mockProductRepo.On("Insert", mock.Anything, &productToInsert).
			Run(func(args mock.Arguments) {
				p := args.Get(1).(*data.Product)
				p.ID = 123
				p.Version = 1
				p.CreatedAt = time.Now()
			}).
			Return(nil)

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		// Asserting the results this way helps to detect data leakage. If instead, we
		// deserialize as structs and use in the assertion, any additional data in the
		// response body will be suppressed during deserialization.
		expectedResponse := `{
			"product": {
				"id": 123,
				"name": "Test Product",
				"category_id": 1,
				"description": "A test product",
				"price": 19.99,
				"quantity": 10,
				"version": 1
			}
		}`

		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
		assert.Equal(t, "/v1/api/products/123", res.Header.Get("Location"))
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("create product successfully with only required fields", func(t *testing.T) {
		input := `{
			"name": "Test Product",
			"category_id": 1
		}`

		mockProduct := data.Product{
			Name:       "Test Product",
			CategoryID: 1,
		}
		rw, req, h, mockProductRepo := setupProductHandlerTest(t, &buf, strings.NewReader(input))
		mockProductRepo.On("Insert", mock.Anything, &mockProduct).
			Run(func(args mock.Arguments) {
				p := args.Get(1).(*data.Product)
				p.ID = 123
				p.Version = 1
				p.CreatedAt = time.Now()
			}).
			Return(nil)

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"product": {
				"id": 123,
				"name": "Test Product",
				"category_id": 1,
				"description": "",
				"price": 0,
				"quantity": 0,
				"version": 1
			}
		}`

		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("request body contains unknown field", func(t *testing.T) {
		payload := `{
			"id": 23,
			"name": "Test Product",
			"category_id": 1,
			"description": "A test product",
			"price": 19.99,
			"quantity": 10
		}`

		rw, req, h, mockProductRepo := setupProductHandlerTest(t, &buf, strings.NewReader(payload))
		mockProductRepo.On("Insert", mock.Anything, &productToInsert).Return(nil)

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.JSONEq(
			t,
			`{"error": "json: unknown field \"id\""}`,
			string(body),
		)

		logMsg := "level=ERROR msg=\"json: unknown field \\\"id\\\"\" method=POST uri=/products\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("empty request body", func(t *testing.T) {
		rw, req, h, mockProductRepo := setupProductHandlerTest(t, &buf, strings.NewReader(""))
		mockProductRepo.On("Insert", mock.Anything, &productToInsert).Return(nil)

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)
		assert.JSONEq(
			t,
			`{"error": "body must not be empty"}`,
			string(body),
		)

		logMsg := "level=ERROR msg=\"body must not be empty\" method=POST uri=/products\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("failed validation", func(t *testing.T) {
		payload := `{
			"name": "Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones",
			"description": "A test product",
			"price": -3,
			"quantity": -25
		}`

		rw, req, h, mockProductRepo := setupProductHandlerTest(t, &buf, strings.NewReader(payload))
		mockProductRepo.On("Insert", mock.Anything, &productToInsert).Return(nil)

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(
			t,
			`{
				"error": {
					"category_id": "is required",
					"name": "must be at most 100 characters long",
					"price": "must be greater than or equal to 0",
					"quantity": "must be greater than or equal to 0"
				}
			}`,
			string(body),
		)

		// Assert Log
		logMsg := "level=ERROR msg=\"Key: 'productDTO.Name' Error:Field validation for 'Name' failed on the 'max' tag\\nKey: 'productDTO.CategoryID' Error:Field validation for 'CategoryID' failed on the 'required' tag\\nKey: 'productDTO.Price' Error:Field validation for 'Price' failed on the 'gte' tag\\nKey: 'productDTO.Quantity' Error:Field validation for 'Quantity' failed on the 'gte' tag\" method=POST uri=/products\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("insert query not found error", func(t *testing.T) {
		rw, req, h, mockProductRepo := setupProductHandlerTest(t, &buf, strings.NewReader(payload))
		mockProductRepo.On("Insert", mock.Anything, &productToInsert).
			Return(data.ErrInvalidCategoryId)

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(
			t,
			`{"error":"invalid category_id"}`,
			string(body),
		)

		// Assert Log
		logMsg := "level=ERROR msg=\"invalid category_id\" method=POST uri=/products\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("insert query server error", func(t *testing.T) {
		rw, req, h, mockProductRepo := setupProductHandlerTest(t, &buf, strings.NewReader(payload))
		mockProductRepo.On("Insert", mock.Anything, &productToInsert).
			Return(errors.New("insert query error"))

		h.CreateProductHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(
			t,
			`{"error": "the server encountered a problem and could not process your request"}`,
			string(body),
		)

		// Assert Log
		logMsg := "level=ERROR msg=\"insert query error\" method=POST uri=/products\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})
}

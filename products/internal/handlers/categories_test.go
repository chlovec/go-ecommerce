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

type MockCategoryRepository struct {
	mock.Mock
}

func (m *MockCategoryRepository) Insert(ctx context.Context, category *data.Category) error {
	args := m.Called(ctx, category)
	return args.Error(0)
}

func setupCategoryHandlerTest(
	t *testing.T,
	w io.Writer,
	body io.Reader,
) (*httptest.ResponseRecorder, *http.Request, Handlers, *MockCategoryRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(w, nil))
	req := httptest.NewRequest(http.MethodPost, "/categories", body)
	rw := httptest.NewRecorder()
	mockCategoryRepo := new(MockCategoryRepository)

	handlers := Handlers{
		logger:    logger,
		validator: validator.New(),
		models: data.Models{
			Category: mockCategoryRepo,
		},
	}

	return rw, req, handlers, mockCategoryRepo
}

func TestCategoryHandler_CreateCategoryHandler(t *testing.T) {
	var buf bytes.Buffer
	payload := `{
		"name": "Test Category",
		"description": "A test category"
	}`

	categoryToInsert := data.Category{
		Name:        "Test Category",
		Description: "A test category",
	}

	t.Run("create category successfully", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			strings.NewReader(payload),
		)
		mockCategoryRepo.On("Insert", mock.Anything, &categoryToInsert).
			Run(func(args mock.Arguments) {
				p := args.Get(1).(*data.Category)
				p.ID = 123
				p.Version = 1
				p.CreatedAt = time.Now()
			}).
			Return(nil)

		h.CreateCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"category": {
				"id": 123,
				"name": "Test Category",
				"description": "A test category",
				"version": 1
			}
		}`

		assert.Equal(t, http.StatusCreated, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("create category successfully with only required fields", func(t *testing.T) {
		payload := `{
			"name": "Test Category"
		}`

		categoryToInsert := data.Category{
			Name: "Test Category",
		}
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			strings.NewReader(payload),
		)
		mockCategoryRepo.On("Insert", mock.Anything, &categoryToInsert).
			Run(func(args mock.Arguments) {
				p := args.Get(1).(*data.Category)
				p.ID = 123
				p.Version = 1
				p.CreatedAt = time.Now()
			}).
			Return(nil)

		h.CreateCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"category": {
				"id": 123,
				"name": "Test Category",
				"description":"",
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
			"name": "Test Category",
			"category_id": 1,
			"description": "A test category",
			"quantity": 10
		}`

		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			strings.NewReader(payload),
		)
		mockCategoryRepo.On("Insert", mock.Anything, &categoryToInsert).Return(nil)

		h.CreateCategoryHandler(rw, req)
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

		logMsg := "level=ERROR msg=\"json: unknown field \\\"id\\\"\" method=POST uri=/categories\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("empty request body", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(t, &buf, strings.NewReader(""))
		mockCategoryRepo.On("Insert", mock.Anything, &categoryToInsert).Return(nil)

		h.CreateCategoryHandler(rw, req)
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

		logMsg := "level=ERROR msg=\"body must not be empty\" method=POST uri=/categories\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("failed validation - required validation", func(t *testing.T) {
		payload := `{
			"name": "",
			"description": "A test category"
		}`

		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			strings.NewReader(payload),
		)
		mockCategoryRepo.On("Insert", mock.Anything, &categoryToInsert).Return(nil)

		h.CreateCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(
			t,
			`{"error": {"name": "is required"}}`,
			string(body),
		)

		// Assert Log
		logMsg := "level=ERROR msg=\"Key: 'categoryDTO.Name' Error:Field validation for 'Name' failed on the 'required' tag\" method=POST uri=/categories\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("failed validation - max length validation", func(t *testing.T) {
		payload := `{
			"name": "Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones Bluetooth Headphones"
		}`

		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			strings.NewReader(payload),
		)
		mockCategoryRepo.On("Insert", mock.Anything, &categoryToInsert).Return(nil)

		h.CreateCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(
			t,
			`{"error": {"name": "must be at most 100 characters long"}}`,
			string(body),
		)

		// Assert Log
		logMsg := "level=ERROR msg=\"Key: 'categoryDTO.Name' Error:Field validation for 'Name' failed on the 'max' tag\" method=POST uri=/categories\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})

	t.Run("insert query server error", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			strings.NewReader(payload),
		)
		mockCategoryRepo.On("Insert", mock.Anything, &categoryToInsert).
			Return(errors.New("insert query error"))

		h.CreateCategoryHandler(rw, req)
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
		logMsg := "level=ERROR msg=\"insert query error\" method=POST uri=/categories\n"
		assert.Contains(t, buf.String(), logMsg)
		buf.Reset()
	})
}

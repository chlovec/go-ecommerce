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
	"github.com/julienschmidt/httprouter"
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

func (m *MockCategoryRepository) GetByID(ctx context.Context, id int64) (*data.Category, error) {
	args := m.Called(ctx, id)
	category, _ := args.Get(0).(*data.Category) // safe type assertion
	return category, args.Error(1)
}

func setupCategoryHandlerTest(
	t *testing.T,
	w io.Writer,
	body io.Reader,
	httpMethod string,
	httpTarget string,
) (*httptest.ResponseRecorder, *http.Request, Handlers, *MockCategoryRepository) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(w, nil))
	req := httptest.NewRequest(httpMethod, httpTarget, body)
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
			http.MethodPost,
			"/categories",
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
			http.MethodPost,
			"/categories",
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
			http.MethodPost,
			"/categories",
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
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			strings.NewReader(""),
			http.MethodPost,
			"/categories",
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
			http.MethodPost,
			"/categories",
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
			http.MethodPost,
			"/categories",
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
			http.MethodPost,
			"/categories",
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

func TestCategoryHandler_GetByID(t *testing.T) {
	var id int64 = 23
	var buf bytes.Buffer

	// Inject httprouter params into request context
	params := httprouter.Params{
		httprouter.Param{Key: "id", Value: "23"},
	}

	t.Run("fetch category successfully", func(t *testing.T) {
		category := data.Category{
			ID:          id,
			Name:        "Test Category",
			Description: "A test category",
			Version:     1,
			CreatedAt:   time.Now(),
		}

		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			nil,
			http.MethodGet,
			"/categories/23",
		)
		req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))
		mockCategoryRepo.On("GetByID", mock.Anything, id).Return(&category, nil)

		h.GetCategoryByID(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"category": {
				"id": 23,
				"name": "Test Category",
				"description": "A test category",
				"version": 1
			}
		}`
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("negative id", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			nil,
			http.MethodGet,
			"/categories/-1",
		)

		invalidParams := httprouter.Params{
			httprouter.Param{Key: "id", Value: "-1"},
		}
		req = req.WithContext(
			context.WithValue(req.Context(), httprouter.ParamsKey, invalidParams),
		)
		mockCategoryRepo.On("GetByID", mock.Anything, id).Return(
			nil, data.ErrRecordNotFound,
		)

		h.GetCategoryByID(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(t, `{"error":"invalid id parameter: -1"}`, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("invalid id type", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			nil,
			http.MethodGet,
			"/categories/abc",
		)

		invalidParams := httprouter.Params{
			httprouter.Param{Key: "id", Value: "abc"},
		}
		req = req.WithContext(
			context.WithValue(req.Context(), httprouter.ParamsKey, invalidParams),
		)
		mockCategoryRepo.On("GetByID", mock.Anything, id).Return(
			nil, data.ErrRecordNotFound,
		)

		h.GetCategoryByID(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(t, `{"error":"invalid id parameter: abc"}`, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("record not found", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			nil,
			http.MethodGet,
			"/categories/23",
		)
		req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))
		mockCategoryRepo.On("GetByID", mock.Anything, id).Return(
			nil, data.ErrRecordNotFound,
		)

		h.GetCategoryByID(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{"error":"the requested resource could not be found"}`
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("record not found", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			nil,
			http.MethodGet,
			"/categories/23",
		)
		req = req.WithContext(context.WithValue(req.Context(), httprouter.ParamsKey, params))
		mockCategoryRepo.On("GetByID", mock.Anything, id).Return(
			nil, errors.New("error processing record"),
		)

		h.GetCategoryByID(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{"error":"the server encountered a problem and could not process your request"}`
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Contains(t, buf.String(), "")
		buf.Reset()
	})
}

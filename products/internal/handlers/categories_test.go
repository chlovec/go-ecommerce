package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
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

func (m *MockCategoryRepository) GetAll(
	ctx context.Context,
	filter data.Filters,
) ([]*data.Category, data.Metadata, error) {
	args := m.Called(ctx, filter)
	categories, _ := args.Get(0).([]*data.Category)
	metadata, _ := args.Get(1).(data.Metadata)
	return categories, metadata, args.Error(2)
}

func setupCategoryHandlerTest(
	t *testing.T,
	w io.Writer,
	body io.Reader,
	httpMethod string,
	httpTarget string,
) (*httptest.ResponseRecorder, *http.Request, Handlers, *MockCategoryRepository) {
	t.Helper()

	logger := slog.New(slog.NewJSONHandler(w, nil))
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

func ParseLog(t *testing.T, buf *bytes.Buffer) map[string]interface{} {
	var logMap map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logMap); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	return logMap
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
		assert.Equal(t, buf.String(), "")
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
		assert.Equal(t, buf.String(), "")
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

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "POST", logData["method"])
		assert.Equal(t, "json: unknown field \"id\"", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "POST", logData["method"])
		assert.Equal(t, "body must not be empty", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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
		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "POST", logData["method"])
		assert.Equal(
			t,
			"Key: 'categoryDTO.Name' Error:Field validation for 'Name' failed on the 'required' tag",
			logData["msg"],
		)
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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
		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "POST", logData["method"])
		assert.Equal(
			t,
			"Key: 'categoryDTO.Name' Error:Field validation for 'Name' failed on the 'max' tag",
			logData["msg"],
		)
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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
		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "POST", logData["method"])
		assert.Equal(t, "insert query error", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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

		h.GetCategoryHandler(rw, req)
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
		assert.Equal(t, buf.String(), "")
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

		h.GetCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(t, `{"error":"invalid id parameter: -1"}`, string(body))

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories/-1", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "GET", logData["method"])
		assert.Equal(t, "invalid id parameter: -1", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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

		h.GetCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		assert.JSONEq(t, `{"error":"invalid id parameter: abc"}`, string(body))

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories/abc", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "GET", logData["method"])
		assert.Equal(t, "invalid id parameter: abc", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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

		h.GetCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusNotFound, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{"error":"the requested resource could not be found"}`
		assert.JSONEq(t, expectedResponse, string(body))

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories/23", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "GET", logData["method"])
		assert.Equal(t, "record not found", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
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

		h.GetCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{"error":"the server encountered a problem and could not process your request"}`
		assert.JSONEq(t, expectedResponse, string(body))

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories/23", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "GET", logData["method"])
		assert.Equal(t, "error processing record", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
		buf.Reset()
	})
}

func TestCategoryHandler_List(t *testing.T) {
	var buf bytes.Buffer

	t.Run("fetch category successfully", func(t *testing.T) {
		category := data.Category{
			ID:          123,
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
			"/categories",
		)

		filters := data.Filters{IDs: []int64{}, Page: 1, PageSize: 20, Sorts: []string{}}
		metadata := data.Metadata{
			CurrentPage:  1,
			PageSize:     20,
			FirstPage:    1,
			LastPage:     4,
			TotalRecords: 65,
		}
		mockCategoryRepo.On("GetAll", mock.Anything, filters).
			Return([]*data.Category{&category}, metadata, nil)

		h.ListCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"categories": [{
				"id": 123,
				"name": "Test Category",
				"description": "A test category",
				"version": 1
			}],
			"metadata":{
				"current_page": 1,
				"page_size": 20,
				"first_page": 1,
				"last_page": 4,
				"page_size": 20,
				"total_records": 65
			}
		}`
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Equal(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("fetch category successfully with query strings", func(t *testing.T) {
		category := data.Category{
			ID:          123,
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
			"/categories",
		)

		q := req.URL.Query()
		q.Set("page", "92")
		q.Set("page_size", "100")
		q.Set("name", "test")
		q.Set("id", "23,92,48,54")
		q.Set("date_from", "2020-01-30T00:00:00Z")
		q.Set("date_to", "2025-08-10T15:04:05Z")
		q.Set("sort", "id,-created_at,-name")
		req.URL.RawQuery = q.Encode()

		dateFrom := time.Date(2020, time.January, 30, 0, 0, 0, 0, time.UTC)
		dateTo := time.Date(2025, time.August, 10, 15, 4, 5, 0, time.UTC)
		filters := data.Filters{
			IDs:      []int64{23, 92, 48, 54},
			Name:     "test",
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
			Page:     92,
			PageSize: 100,
			Sorts:    []string{"id", "-created_at", "-name"},
		}
		metadata := data.Metadata{
			CurrentPage: 92, PageSize: 100, FirstPage: 1, LastPage: 98, TotalRecords: 9701,
		}
		mockCategoryRepo.On(
			"GetAll", mock.Anything, filters,
		).Return([]*data.Category{&category}, metadata, nil)

		h.ListCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"categories": [{
				"id": 123,
				"name": "Test Category",
				"description": "A test category",
				"version": 1
			}],
			"metadata":{
				"current_page": 92,
				"page_size": 100,
				"first_page": 1,
				"last_page": 98,
				"page_size": 100,
				"total_records": 9701
			}
		}`
		assert.JSONEq(t, expectedResponse, string(body))
		assert.Equal(t, buf.String(), "")
		buf.Reset()
	})

	t.Run("error parsing query strings", func(t *testing.T) {
		category := data.Category{
			ID:          123,
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
			"/categories",
		)

		q := req.URL.Query()
		q.Set("page", "as")
		q.Set("page_size", "bk")
		q.Set(
			"name",
			"testyyhhhhanbgdrsebdbdbdbdbdbdbdbd testyyhhhhanbgdrsebdbdbdbdbdbdbdbd testyyhhhhanbgdrsebdbdbdbdbdbdbdbd",
		)
		q.Set("id", "23,92,48,.")
		q.Set("date_from", "2020-01-30")
		q.Set("date_to", "2025-08-10")
		q.Set("sort", "id,-test,-name")
		req.URL.RawQuery = q.Encode()

		dateFrom := time.Date(2020, time.January, 30, 0, 0, 0, 0, time.UTC)
		dateTo := time.Date(2025, time.August, 10, 15, 4, 5, 0, time.UTC)
		filters := data.Filters{
			IDs:      []int64{23, 92, 48, 54},
			Name:     "test",
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
			Page:     92,
			PageSize: 100,
			Sorts:    []string{"id", "-created_at", "-name"},
		}
		metadata := data.Metadata{
			CurrentPage: 92, PageSize: 100, FirstPage: 1, LastPage: 98, TotalRecords: 9701,
		}
		mockCategoryRepo.On(
			"GetAll", mock.Anything, filters,
		).Return([]*data.Category{&category}, metadata, nil)

		h.ListCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusBadRequest, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"error": {
				"date_from": "invalid datetime: 2020-01-30",
				"date_to": "invalid datetime: 2025-08-10",
				"id": "invalid id: \".\"",
				"page": "must be an integer value: as",
				"page_size": "must be an integer value: bk"
			}
		}`
		assert.JSONEq(t, expectedResponse, string(body))

		logData := ParseLog(t, &buf)
		uri := "/categories?date_from=2020-01-30&date_to=2025-08-10&id=23%2C92%2C48%2C.&name=testyyhhhhanbgdrsebdbdbdbdbdbdbdbd+testyyhhhhanbgdrsebdbdbdbdbdbdbdbd+testyyhhhhanbgdrsebdbdbdbdbdbdbdbd&page=as&page_size=bk&sort=id%2C-test%2C-name"
		expectedMsg := []string{
			"date_to invalid datetime: 2025-08-10",
			"id invalid id: \".\"",
			"page must be an integer value: as",
			"page_size must be an integer value: bk",
			"date_from invalid datetime: 2020-01-30",
		}
		actualMsg := strings.Split(logData["msg"].(string), "; ")
		sort.Strings(expectedMsg)
		sort.Strings(actualMsg)

		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, uri, logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "GET", logData["method"])
		assert.Equal(t, expectedMsg, actualMsg)
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
		buf.Reset()
	})

	t.Run("query string validation error", func(t *testing.T) {
		category := data.Category{
			ID:          123,
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
			"/categories",
		)

		q := req.URL.Query()
		q.Set("page", "-10")
		q.Set("page_size", "103")
		q.Set(
			"name",
			"testyyhhhhanbgdrsebdbdbdbdbdbdbdbd testyyhhhhanbgdrsebdbdbdbdbdbdbdbd testyyhhhhanbgdrsebdbdbdbdbdbdbdbd",
		)
		q.Set("id", "23,92,48")
		q.Set("date_from", "2020-01-30T00:00:00Z")
		q.Set("date_to", "2025-08-10T15:04:05Z")
		q.Set("sort", "id,-test,-name")
		req.URL.RawQuery = q.Encode()

		dateFrom := time.Date(2020, time.January, 30, 0, 0, 0, 0, time.UTC)
		dateTo := time.Date(2025, time.August, 10, 15, 4, 5, 0, time.UTC)
		filters := data.Filters{
			IDs:      []int64{23, 92, 48, 54},
			Name:     "test",
			DateFrom: &dateFrom,
			DateTo:   &dateTo,
			Page:     92,
			PageSize: 100,
			Sorts:    []string{"id", "-created_at", "-name"},
		}
		metadata := data.Metadata{
			CurrentPage: 92, PageSize: 100, FirstPage: 1, LastPage: 98, TotalRecords: 9701,
		}
		mockCategoryRepo.On(
			"GetAll", mock.Anything, filters,
		).Return([]*data.Category{&category}, metadata, nil)

		h.ListCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusUnprocessableEntity, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{
			"error": {
				"name": "must be greater than or equal to 1",
				"name": "must be at most 100 characters long",
				"Sorts[1]": "must be one of [id created_at name -id -created_at -name]",
				"page_size": "must be less than or equal to 100",
				"page": "must be greater than or equal to 1"
			}
		}`
		assert.JSONEq(t, expectedResponse, string(body))

		uri := "/categories?date_from=2020-01-30T00%3A00%3A00Z&date_to=2025-08-10T15%3A04%3A05Z&id=23%2C92%2C48&name=testyyhhhhanbgdrsebdbdbdbdbdbdbdbd+testyyhhhhanbgdrsebdbdbdbdbdbdbdbd+testyyhhhhanbgdrsebdbdbdbdbdbdbdbd&page=-10&page_size=103&sort=id%2C-test%2C-name"
		msg := "Key: 'Filters.Name' Error:Field validation for 'Name' failed on the 'max' tag\nKey: 'Filters.Sorts[1]' Error:Field validation for 'Sorts[1]' failed on the 'oneof' tag\nKey: 'Filters.Page' Error:Field validation for 'Page' failed on the 'gte' tag\nKey: 'Filters.PageSize' Error:Field validation for 'PageSize' failed on the 'lte' tag"

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, uri, logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "GET", logData["method"])
		assert.Equal(t, msg, logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
		buf.Reset()
	})

	t.Run("db error", func(t *testing.T) {
		rw, req, h, mockCategoryRepo := setupCategoryHandlerTest(
			t,
			&buf,
			nil,
			http.MethodGet,
			"/categories",
		)

		filters := data.Filters{IDs: []int64{}, Page: 1, PageSize: 20, Sorts: []string{}}
		mockCategoryRepo.On("GetAll", mock.Anything, filters).
			Return([]*data.Category{}, data.Metadata{}, errors.New("db error"))

		h.ListCategoryHandler(rw, req)
		res := rw.Result()
		defer res.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
		assert.Equal(t, "application/json", res.Header.Get("Content-Type"))

		body, err := io.ReadAll(res.Body)
		assert.NoError(t, err)

		expectedResponse := `{"error": "the server encountered a problem and could not process your request"}`
		assert.JSONEq(t, expectedResponse, string(body))

		logData := ParseLog(t, &buf)
		assert.Equal(t, "ERROR", logData["level"])
		assert.Equal(t, "/categories", logData["uri"])
		assert.NotEqual(t, "", logData["time"])
		assert.Equal(t, "GET", logData["method"])
		assert.Equal(t, "db error", logData["msg"])
		assert.Equal(t, 5, len(logData), "expected 5 entries, got %d", len(logData))
		buf.Reset()
	})
}

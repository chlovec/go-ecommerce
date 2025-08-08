package data

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCategoryModel_Insert(t *testing.T) {
	t.Parallel()

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	categoryModel := CategoryModel{DB: db}
	ctx := context.Background()

	mockQuery := regexp.QuoteMeta(`
		INSERT INTO categories(name, description)
		VALUES($1, $2)
		RETURNING id, created_at, version
	`)

	t.Run("success", func(t *testing.T) {
		category := Category{
			Name:        "Test Category",
			Description: "A test category",
		}

		createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
		sqlMock.ExpectQuery(mockQuery).
			WithArgs(category.Name, category.Description).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "version"}).
				AddRow(1, createdAt, 1))

		expectedCategory := Category{
			ID:          1,
			Name:        "Test Category",
			Description: "A test category",
			Version:     1,
			CreatedAt:   createdAt,
		}

		err := categoryModel.Insert(ctx, &category)

		assert.NoError(t, err)
		assert.Equal(t, expectedCategory, category)
		assert.Equal(t, 1, category.Version)
	})

	t.Run("other error", func(t *testing.T) {
		category := Category{
			Name:        "Test Category",
			Description: "A test category",
		}

		dbErr := errors.New("unexpected DB error")
		sqlMock.ExpectQuery(mockQuery).
			WithArgs(category.Name, category.Description).
			WillReturnError(dbErr)

		err := categoryModel.Insert(ctx, &category)
		assert.Equal(t, dbErr, err)
	})

	t.Run("context canceled", func(t *testing.T) {
		category := Category{
			Name:        "Test Category",
			Description: "A test category",
		}

		newCtx, cancel := context.WithCancel(context.Background())
		cancel()

		sqlMock.ExpectQuery(mockQuery).
			WithArgs(category.Name, category.Description).
			WillReturnError(errors.New("unexpected DB error"))

		err := categoryModel.Insert(newCtx, &category)
		assert.Equal(t, "context canceled", err.Error())
	})
}

func TestCategoryModel_GetByID(t *testing.T) {
	t.Parallel()

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	categoryModel := CategoryModel{DB: db}
	ctx := context.Background()

	var mockQuery = regexp.QuoteMeta(`
		SELECT id, name, description, created_at, version
		FROM categories
		WHERE id = $1
	`)

	t.Run("returns category with the given id", func(t *testing.T) {
		var id int64 = 23
		createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
		expected := Category{
			ID:          id,
			Name:        "Test Category",
			Description: "A test category",
			Version:     1,
			CreatedAt:   createdAt,
		}

		mockRow := sqlmock.NewRows(
			[]string{"id", "name", "description", "created_at", "version"},
		).AddRow(id, "Test Category", "A test category", createdAt, 1)
		sqlMock.ExpectQuery(mockQuery).WithArgs(id).WillReturnRows(mockRow)

		actual, err := categoryModel.GetByID(ctx, id)

		assert.NoError(t, err)
		assert.Equal(t, expected, *actual)
	})

	t.Run("no rows returned", func(t *testing.T) {
		mockRow := sqlmock.NewRows(
			[]string{"id", "name", "description", "created_at", "version"},
		)
		sqlMock.ExpectQuery(mockQuery).WithArgs(23).WillReturnRows(mockRow)

		actual, err := categoryModel.GetByID(ctx, 23)

		assert.Error(t, err)
		assert.Equal(t, ErrRecordNotFound, err)
		assert.Nil(t, actual)
	})

	t.Run("no rows returned", func(t *testing.T) {
		mockError := errors.New("db error")
		sqlMock.ExpectQuery(mockQuery).WithArgs(23).WillReturnError(mockError)

		actual, err := categoryModel.GetByID(ctx, 23)

		assert.Error(t, err)
		assert.Equal(t, mockError, err)
		assert.Nil(t, actual)
	})

	t.Run("context canceled", func(t *testing.T) {
		newCtx, cancel := context.WithCancel(context.Background())
		cancel()

		mockError := errors.New("db error")
		sqlMock.ExpectQuery(mockQuery).WithArgs(23).WillReturnError(mockError)

		actual, err := categoryModel.GetByID(newCtx, 23)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		assert.Nil(t, actual)
	})
}

func TestUpdateCategoryModel_Update(t *testing.T) {
	t.Parallel()

	mockQuery := regexp.QuoteMeta(`
		UPDATE categories 
		SET name = $1, description = $2, version = version + 1
		WHERE id = $3 AND version = $4
		RETURNING version
	`)

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	categoryModel := CategoryModel{DB: db}
	ctx := context.Background()

	createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
	t.Run("update category successfully", func(t *testing.T) {
		category := Category{
			ID:          1,
			Name:        "Test Category",
			Description: "A test category",
			Version:     1,
			CreatedAt:   createdAt,
		}

		sqlMock.ExpectQuery(mockQuery).WithArgs(
			category.Name, category.Description, category.ID, category.Version,
		).WillReturnRows(sqlmock.NewRows([]string{"version"}).AddRow(2))

		err := categoryModel.Update(ctx, &category)

		expectedCategory := Category{
			ID:          1,
			Name:        "Test Category",
			Description: "A test category",
			Version:     2,
			CreatedAt:   createdAt,
		}
		assert.NoError(t, err)
		assert.Equal(t, expectedCategory, category)
	})

	t.Run("edit conflict error", func(t *testing.T) {
		category := Category{
			ID:          1,
			Name:        "Test Category",
			Description: "A test category",
			Version:     1,
			CreatedAt:   createdAt,
		}

		sqlMock.ExpectQuery(mockQuery).WithArgs(
			category.Name, category.Description, category.ID, category.Version,
		).WillReturnError(sql.ErrNoRows)

		err := categoryModel.Update(ctx, &category)

		assert.Error(t, err)
		assert.Equal(t, err.Error(), "edit conflict")
		assert.Equal(t, category.Version, 1)
	})

	t.Run("update error", func(t *testing.T) {
		category := Category{
			ID:          1,
			Name:        "Test Category",
			Description: "A test category",
			Version:     1,
			CreatedAt:   createdAt,
		}

		sqlMock.ExpectQuery(mockQuery).WithArgs(
			category.Name, category.Description, category.ID, category.Version,
		).WillReturnError(errors.New("db update error"))

		err := categoryModel.Update(ctx, &category)

		assert.Error(t, err)
		assert.Equal(t, err.Error(), "db update error")
		assert.Equal(t, category.Version, 1)
	})

	t.Run("context canceled", func(t *testing.T) {
		category := Category{
			ID:          1,
			Name:        "Test Category",
			Description: "A test category",
			Version:     1,
			CreatedAt:   createdAt,
		}

		newCtx, cancel := context.WithCancel(context.Background())
		cancel()

		sqlMock.ExpectQuery(mockQuery).WithArgs(
			category.Name, category.Description, category.ID, category.Version,
		).WillReturnError(errors.New("db update error"))

		err := categoryModel.Update(newCtx, &category)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestCategoryModel_Delete(t *testing.T) {
	t.Parallel()

	var id int64 = 1
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	categoryModel := CategoryModel{DB: db}
	ctx := context.Background()

	mockQuery := regexp.QuoteMeta(`DELETE FROM categories WHERE id = $1`)

	t.Run("delete success", func(t *testing.T) {
		sqlMock.ExpectExec(mockQuery).WithArgs(id).WillReturnResult(
			sqlmock.NewResult(1, 1),
		)
		err := categoryModel.Delete(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("delete error", func(t *testing.T) {
		sqlMock.ExpectExec(mockQuery).WithArgs(id).WillReturnError(
			errors.New("delete error"),
		)
		err := categoryModel.Delete(ctx, id)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "delete error")
	})

	t.Run("zero rows affected", func(t *testing.T) {
		sqlMock.ExpectExec(mockQuery).WithArgs(id).WillReturnResult(
			sqlmock.NewResult(1, 0),
		)
		err := categoryModel.Delete(ctx, id)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "record not found")
	})

	t.Run("rows affected error", func(t *testing.T) {
		sqlMock.ExpectExec(mockQuery).WithArgs(id).WillReturnResult(
			sqlmock.NewErrorResult(errors.New("rows affected error")),
		)
		err := categoryModel.Delete(ctx, id)
		assert.Error(t, err)
		assert.Equal(t, err.Error(), "rows affected error")
	})

	t.Run("context canceled", func(t *testing.T) {
		newCtx, cancel := context.WithCancel(context.Background())
		cancel()

		sqlMock.ExpectExec(mockQuery).
			WithArgs(int64(123)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := categoryModel.Delete(newCtx, 123)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})
}

func TestCategoryModel_GetAll(t *testing.T) {
	t.Parallel()

	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	categoryModel := CategoryModel{DB: db}
	ctx := context.Background()

	filters := Filters{
		Page:     1,
		PageSize: 20,
		Sort:     "",
	}

	t.Run("fetch all categories successfully", func(t *testing.T) {
		createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
		mockQuery := regexp.QuoteMeta(`
			SELECT count(*) OVER(), id, name, description, created_at, version
			FROM categories
			WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
			ORDER BY id ASC
			Limit $2 OFFSET $3`,
		)
		mockRow := sqlmock.NewRows(
			[]string{"total_pages", "id", "name", "description", "created_at", "version"},
		).AddRow(10, 121, "Test Category", "A test category", createdAt, 1)
		sqlMock.ExpectQuery(mockQuery).WithArgs("", 20, 0).WillReturnRows(mockRow)

		categories, metadata, err := categoryModel.GetAll(ctx, "", filters)

		testCategory := Category{
			ID:          121,
			Name:        "Test Category",
			Description: "A test category",
			CreatedAt:   createdAt,
			Version:     1,
		}
		expectedMetadata := Metadata{
			CurrentPage:  1,
			PageSize:     20,
			FirstPage:    1,
			LastPage:     1,
			TotalRecords: 10,
		}
		expectedCategories := []*Category{&testCategory}
		assert.NoError(t, err)
		assert.Equal(t, expectedCategories, categories)
		assert.Equal(t, expectedMetadata, metadata)
	})

	t.Run("filter and sort categories by name desc", func(t *testing.T) {
		currentFilters := Filters{
			Page:     1,
			PageSize: 20,
			Sort:     "-name",
		}
		createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
		mockQuery := regexp.QuoteMeta(`
			SELECT count(*) OVER(), id, name, description, created_at, version
			FROM categories
			WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
			ORDER BY name DESC, id ASC
			Limit $2 OFFSET $3`,
		)
		mockRow := sqlmock.NewRows(
			[]string{"total_pages", "id", "name", "description", "created_at", "version"},
		).AddRow(1003, 121, "Test Category", "A test category", createdAt, 1)
		sqlMock.ExpectQuery(mockQuery).WithArgs("test", 20, 0).WillReturnRows(mockRow)

		categories, metadata, err := categoryModel.GetAll(ctx, "test", currentFilters)

		testCategory := Category{
			ID:          121,
			Name:        "Test Category",
			Description: "A test category",
			CreatedAt:   createdAt,
			Version:     1,
		}
		expectedMetadata := Metadata{
			CurrentPage:  1,
			PageSize:     20,
			FirstPage:    1,
			LastPage:     51,
			TotalRecords: 1003,
		}
		expectedCategories := []*Category{&testCategory}
		assert.NoError(t, err)
		assert.Equal(t, expectedCategories, categories)
		assert.Equal(t, expectedMetadata, metadata)
	})

	t.Run("filter and sort categories by name asc", func(t *testing.T) {
		currentFilters := Filters{
			Page:     1,
			PageSize: 20,
			Sort:     "name",
		}
		mockQuery := regexp.QuoteMeta(`
			SELECT count(*) OVER(), id, name, description, created_at, version
			FROM categories
			WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
			ORDER BY name ASC, id ASC
			Limit $2 OFFSET $3`,
		)
		mockRow := sqlmock.NewRows(
			[]string{"total_pages", "id", "name", "description", "created_at", "version"},
		)
		sqlMock.ExpectQuery(mockQuery).WithArgs("test", 20, 0).WillReturnRows(mockRow)
		categories, metadata, err := categoryModel.GetAll(ctx, "test", currentFilters)
		assert.NoError(t, err)
		assert.Equal(t, []*Category{}, categories)
		assert.Equal(t, Metadata{}, metadata)
	})

	t.Run("error executing query", func(t *testing.T) {
		mockQuery := regexp.QuoteMeta(
			`SELECT count(*) OVER(), id, name, description, created_at, version
			FROM categories
			WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
			ORDER BY id ASC
			Limit $2 OFFSET $3`,
		)

		queryErr := "error executing query"
		sqlMock.ExpectQuery(mockQuery).WithArgs("test", 20, 0).WillReturnError(
			errors.New(queryErr),
		)

		categories, metadata, err := categoryModel.GetAll(ctx, "test", filters)
		assert.Error(t, err)
		assert.Equal(t, queryErr, err.Error())
		assert.Nil(t, categories)
		assert.Equal(t, Metadata{}, metadata)
	})

	t.Run("scan error", func(t *testing.T) {
		mockQuery := regexp.QuoteMeta(
			`SELECT count(*) OVER(), id, name, description, created_at, version
			FROM categories
			WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
			ORDER BY id ASC
			Limit $2 OFFSET $3`,
		)
		mockRow := sqlmock.NewRows(
			[]string{"total_pages", "id", "name", "description", "version"},
		).AddRow(10, 121, "Test Category", "A test category", 1)
		sqlMock.ExpectQuery(mockQuery).WithArgs("", 20, 0).WillReturnRows(mockRow)

		categories, metadata, err := categoryModel.GetAll(ctx, "", filters)
		assert.Error(t, err)
		assert.Equal(t, "sql: expected 5 destination arguments in Scan, not 6", err.Error())
		assert.Nil(t, categories)
		assert.Equal(t, Metadata{}, metadata)
	})

	t.Run("row error", func(t *testing.T) {
		mockQuery := regexp.QuoteMeta(
			`SELECT count(*) OVER(), id, name, description, created_at, version
			FROM categories
			WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
			ORDER BY id ASC
			Limit $2 OFFSET $3`,
		)
		mockRow := sqlmock.NewRows(
			[]string{"count", "id", "name", "description", "created_at", "version"},
		).
			AddRow(1, 1, "Test", "Description", "2025-01-01", 1).
			RowError(0, errors.New("rows iteration error"))

			// Fail on first row

		sqlMock.ExpectQuery(mockQuery).WithArgs("", 20, 0).WillReturnRows(mockRow)

		categories, metadata, err := categoryModel.GetAll(ctx, "", filters)
		assert.Error(t, err)
		assert.Equal(t, "rows iteration error", err.Error())
		assert.Nil(t, categories)
		assert.Equal(t, Metadata{}, metadata)
	})
}

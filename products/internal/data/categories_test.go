package data

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCategoryModel_Insert(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	model := CategoryModel{DB: db}
	ctx := context.Background()

	var expectedQuery = regexp.QuoteMeta(`
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
		mock.ExpectQuery(expectedQuery).
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

		err := model.Insert(ctx, &category)
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
		mock.ExpectQuery(expectedQuery).
			WithArgs(category.Name, category.Description).
			WillReturnError(dbErr)

		err := model.Insert(ctx, &category)
		assert.Equal(t, dbErr, err)
	})
}

func TestCategoryModel_GetByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	model := CategoryModel{DB: db}
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
		mock.ExpectQuery(mockQuery).WithArgs(id).WillReturnRows(mockRow)
		actual, err := model.GetByID(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, expected, *actual)
	})

	t.Run("no rows returned", func(t *testing.T) {
		mockRow := sqlmock.NewRows(
			[]string{"id", "name", "description", "created_at", "version"},
		)
		mock.ExpectQuery(mockQuery).WithArgs(23).WillReturnRows(mockRow)
		actual, err := model.GetByID(ctx, 23)
		assert.Error(t, err)
		assert.Equal(t, ErrRecordNotFound, err)
		assert.Nil(t, actual)
	})

	t.Run("no rows returned", func(t *testing.T) {
		mockError := errors.New("db error")
		mock.ExpectQuery(mockQuery).WithArgs(23).WillReturnError(mockError)
		actual, err := model.GetByID(ctx, 23)
		assert.Error(t, err)
		assert.Equal(t, mockError, err)
		assert.Nil(t, actual)
	})
}

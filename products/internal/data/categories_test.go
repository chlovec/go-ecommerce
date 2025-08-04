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

	// Setup DB mock
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

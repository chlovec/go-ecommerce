package data

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestProductModel_Insert(t *testing.T) {
	t.Parallel()

	// Setup DB mock
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	model := ProductModel{DB: db}
	ctx := context.Background()

	product := Product{
		Name:        "Test Product",
		CategoryID:  999, // doesn't matter for success case
		Description: "A test product",
		Price:       10.99,
		Quantity:    5,
	}

	t.Run("success", func(t *testing.T) {
		productInsert := Product{
			Name:        "Test Product",
			CategoryID:  999, // doesn't matter for success case
			Description: "A test product",
			Price:       10.99,
			Quantity:    5,
		}

		createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
		mock.ExpectQuery("INSERT INTO products").
			WithArgs(productInsert.Name, productInsert.CategoryID, productInsert.Description, productInsert.Price, productInsert.Quantity).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "version"}).
				AddRow(1, createdAt, 1))

		expectedProduct := Product{
			ID:          1,
			Name:        "Test Product",
			CategoryID:  999, // doesn't matter for success case
			Description: "A test product",
			Price:       10.99,
			Quantity:    5,
			Version:     1,
			CreatedAt:   createdAt,
		}

		err := model.Insert(ctx, &productInsert)
		assert.NoError(t, err)
		assert.Equal(t, expectedProduct, productInsert)
		assert.Equal(t, 1, productInsert.Version)
	})

	t.Run("foreign key violation", func(t *testing.T) {
		mock.ExpectQuery("INSERT INTO products").
			WithArgs(product.Name, product.CategoryID, product.Description, product.Price, product.Quantity).
			WillReturnError(&pq.Error{Code: "23503"})

		err := model.Insert(ctx, &product)
		assert.True(t, errors.Is(err, ErrInvalidCategoryId))
		assert.Contains(
			t,
			err.Error(),
			fmt.Sprintf("category_id %d does not exist", product.CategoryID),
		)
	})

	t.Run("other error", func(t *testing.T) {
		dbErr := errors.New("unexpected DB error")
		mock.ExpectQuery("INSERT INTO products").
			WithArgs(product.Name, product.CategoryID, product.Description, product.Price, product.Quantity).
			WillReturnError(dbErr)

		err := model.Insert(ctx, &product)
		assert.Equal(t, dbErr, err)
	})
}

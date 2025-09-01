package data

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestProductModel_Insert(t *testing.T) {
	t.Parallel()

	// Setup DB sqlMock
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	productModel := ProductModel{DB: db}
	ctx := context.Background()

	product := Product{
		Name:        "Test Product",
		CategoryID:  999, // doesn't matter for success case
		Description: "A test product",
		Price:       10.99,
		Quantity:    5,
	}

	var expectedQuery = regexp.QuoteMeta(`
		INSERT INTO products (name,category_id,description,price,quantity) 
		VALUES (?,?,?,?,?)
		RETURNING id, created_at, version
	`)

	t.Run("success", func(t *testing.T) {
		productInsert := Product{
			Name:        "Test Product",
			CategoryID:  999, // doesn't matter for success case
			Description: "A test product",
			Price:       10.99,
			Quantity:    5,
		}

		createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
		mockRow := sqlmock.NewRows(
			[]string{"id", "created_at", "version"},
		).
			AddRow(1, createdAt, 1)
		sqlMock.ExpectQuery(expectedQuery).
			WithArgs(
				productInsert.Name,
				productInsert.CategoryID,
				productInsert.Description,
				productInsert.Price,
				productInsert.Quantity,
			).
			WillReturnRows(mockRow)

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

		err := productModel.Insert(ctx, &productInsert)
		assert.NoError(t, err)
		assert.Equal(t, expectedProduct, productInsert)
		assert.Equal(t, 1, productInsert.Version)
	})

	t.Run("foreign key violation", func(t *testing.T) {
		sqlMock.ExpectQuery(expectedQuery).
			WithArgs(product.Name, product.CategoryID, product.Description, product.Price, product.Quantity).
			WillReturnError(&pq.Error{Code: "23503"})

		err := productModel.Insert(ctx, &product)
		assert.True(t, errors.Is(err, ErrInvalidCategoryId))
		assert.Contains(
			t,
			err.Error(),
			fmt.Sprintf("category_id %d does not exist", product.CategoryID),
		)
	})

	t.Run("other error", func(t *testing.T) {
		dbErr := errors.New("unexpected DB error")
		sqlMock.ExpectQuery(expectedQuery).
			WithArgs(product.Name, product.CategoryID, product.Description, product.Price, product.Quantity).
			WillReturnError(dbErr)

		err := productModel.Insert(ctx, &product)
		assert.Equal(t, dbErr, err)
	})
}

func TestProductModel_GetById(t *testing.T) {
	t.Parallel()

	// Setup DB mock
	db, sqlMock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	productModel := ProductModel{DB: db}
	ctx := context.Background()

	var mockQuery = regexp.QuoteMeta(`
		SELECT id, name, category_id, description, price, quantity, created_at, version
		FROM products
		WHERE id = ?
	`)

	createdAt := time.Date(2023, time.July, 1, 10, 0, 0, 0, time.UTC)
	expectedProduct := Product{
		ID:          1,
		Name:        "Test Product",
		CategoryID:  999, // doesn't matter for success case
		Description: "A test product",
		Price:       10.99,
		Quantity:    5,
		CreatedAt:   createdAt,
		Version:     1,
	}

	t.Run("returns product with the given id", func(t *testing.T) {
		var id int64 = 1
		mockRow := sqlMock.NewRows(
			[]string{
				"id",
				"name",
				"category_id",
				"description",
				"price",
				"quantity",
				"created_at",
				"version",
			},
		).
			AddRow(id, "Test Product", 999, "A test product", 10.99, 5, createdAt, 1)
		sqlMock.ExpectQuery(mockQuery).WithArgs(id).WillReturnRows(mockRow)

		actualProduct, err := productModel.GetByID(ctx, id)
		assert.NoError(t, err)
		assert.Equal(t, expectedProduct, *actualProduct)
	})

	t.Run("no rows returned", func(t *testing.T) {
		var id int64 = 1
		mockRow := sqlMock.NewRows(
			[]string{
				"id",
				"name",
				"category_id",
				"description",
				"price",
				"quantity",
				"created_at",
				"version",
			},
		)
		sqlMock.ExpectQuery(mockQuery).WithArgs(id).WillReturnRows(mockRow)

		actualProduct, err := productModel.GetByID(ctx, id)
		assert.Nil(t, actualProduct)
		assert.Equal(t, ErrRecordNotFound, err)
		assert.Error(t, err)
	})

	t.Run("db error", func(t *testing.T) {
		mockError := errors.New("db error")
		sqlMock.ExpectQuery(mockQuery).WithArgs(1).WillReturnError(mockError)

		actualProduct, err := productModel.GetByID(ctx, 1)
		assert.Nil(t, actualProduct)
		assert.Equal(t, mockError, err)
		assert.Error(t, err)
	})

	t.Run("context canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		mockError := errors.New("db error")
		sqlMock.ExpectQuery(mockQuery).WithArgs(23).WillReturnError(mockError)

		actualProduct, err := productModel.GetByID(ctx, 1)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
		assert.Nil(t, actualProduct)
	})
}

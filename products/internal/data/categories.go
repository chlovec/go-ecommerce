package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"-"`
}

type CategoryModel struct {
	DB *sql.DB
}

type CategoryRepository interface {
	Insert(ctx context.Context, category *Category) error
	GetByID(ctx context.Context, id int64) (*Category, error)
	GetAll(ctx context.Context, filters Filters) ([]*Category, Metadata, error)
}

func (c *CategoryModel) Insert(ctx context.Context, category *Category) error {
	query := `
		INSERT INTO categories(name, description)
		VALUES($1, $2)
		RETURNING id, created_at, version
	`
	args := []any{category.Name, category.Description}
	return c.DB.QueryRowContext(ctx, query, args...).Scan(
		&category.ID,
		&category.CreatedAt,
		&category.Version,
	)
}

func (c *CategoryModel) GetByID(ctx context.Context, id int64) (*Category, error) {
	query := `
		SELECT id, name, description, created_at, version
		FROM categories
		WHERE id = $1
	`
	var category Category
	err := c.DB.QueryRowContext(ctx, query, id).Scan(
		&category.ID,
		&category.Name,
		&category.Description,
		&category.CreatedAt,
		&category.Version,
	)

	// Handle any errors. If there was no record found, Scan()
	// will return a sql.ErrNoRows error. Check for this and
	// return the custom ErrRecordNotFound
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return &category, nil
}

func (c *CategoryModel) Update(ctx context.Context, category *Category) error {
	query := `
		UPDATE categories 
		SET name = $1, description = $2, version = version + 1
		WHERE id = $3 AND version = $4
		RETURNING version
	`

	// Version in the where clause is used for optimistic concurrency. if there is an
	// edit conflict, it will result in sql.ErrNoRows
	args := []any{category.Name, category.Description, category.ID, category.Version}
	err := c.DB.QueryRowContext(ctx, query, args...).Scan(&category.Version)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

// Method for deleting a specific category record.
func (c *CategoryModel) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM categories WHERE id = $1`

	// Execute SQL query using the Exec() method, passing in the id variable as
	// the value for the placeholder parameter. The Exec() method returns a sql.Result
	// value
	result, err := c.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (c *CategoryModel) GetAll(
	ctx context.Context,
	filters Filters,
) ([]*Category, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, name, description, created_at, version
		FROM categories
		WHERE
			(cardinality($1::bigint[]) = 0 OR id = ANY($1))
			AND ($2 = '' OR to_tsvector('simple', name) @@ plainto_tsquery('simple', $2))
			AND ($3::timestamp IS NULL OR created_at >= $3)
			AND ($4::timestamp IS NULL OR created_at <= $4)
		ORDER BY %s
		Limit $5 OFFSET $6`,
		filters.sortColumns())

	args := []any{
		pq.Array(filters.IDs),
		filters.Name,
		filters.DateFrom,
		filters.DateTo,
		filters.PageSize,
		filters.offset(),
	}

	rows, err := c.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	categories := []*Category{}
	totalRecords := 0

	for rows.Next() {
		// Initialize an empty category struct to hold the data for an individual category.
		var category Category

		// Scan the values from the row into the categories struct.
		err := rows.Scan(
			&totalRecords,
			&category.ID,
			&category.Name,
			&category.Description,
			&category.CreatedAt,
			&category.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		// Add the category struct to the slice.
		categories = append(categories, &category)
	}

	// After the rows.Next() loop has finished, call rows.Err() to retrieve any error
	// that was encountered during the iteration.
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	// If everything went OK, then return the slice of categories.
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return categories, metadata, nil
}

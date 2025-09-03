package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
)

type Product struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	CategoryID  int       `json:"category_id"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Quantity    int       `json:"quantity"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"-"`
}

type ProductModel struct {
	db *sql.DB
}

type ProductRepository interface {
	Insert(ctx context.Context, product *Product) error
}

func NewProductModel(db *sql.DB) *ProductModel {
	return &ProductModel{db: db}
}

func (p *ProductModel) Insert(ctx context.Context, product *Product) error {
	query, args, _ := sq.Insert("products").
		Columns("name", "category_id", "description", "price", "quantity").
		Values(
			product.Name,
			product.CategoryID,
			product.Description,
			product.Price,
			product.Quantity).
		Suffix("RETURNING id, created_at, version").
		ToSql()
	err := p.db.QueryRowContext(ctx, query, args...).Scan(
		&product.ID,
		&product.CreatedAt,
		&product.Version,
	)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == ErrForeignKeyViolation {
			return fmt.Errorf(
				"category_id %d does not exist: %w",
				product.CategoryID,
				ErrInvalidCategoryId,
			)
		}
		return err
	}

	return nil
}

func (p *ProductModel) GetByID(ctx context.Context, id int64) (*Product, error) {
	query, _, _ := sq.Select(
		"id",
		"name",
		"category_id",
		"description",
		"price",
		"quantity",
		"created_at",
		"version",
	).
		From("products").
		Where(sq.Eq{"id": id}).
		ToSql()

	var product Product
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.CategoryID,
		&product.Description,
		&product.Price,
		&product.Quantity,
		&product.CreatedAt,
		&product.Version,
	)

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRecordNotFound
	} else if err != nil {
		return nil, err
	}

	return &product, nil
}

func (p *ProductModel) GetAll(ctx context.Context, filters Filters) ([]*Product, Metadata, error) {
	builder := sq.Select(
		"id",
		"name",
		"category_id",
		"description",
		"price",
		"quantity",
		"created_at",
		"version",
	).From("products")

	builder = p.buildFilters(builder, filters)
	query, args, _ := builder.ToSql()
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	products := []*Product{}
	for rows.Next() {
		var product Product
		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.CategoryID,
			&product.Description,
			&product.Price,
			&product.Quantity,
			&product.CreatedAt,
			&product.Version,
		); err != nil {
			return nil, Metadata{}, err
		}
		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	totalRecords, err := p.countProducts(ctx, filters)
	if err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(int(totalRecords), filters.Page, filters.PageSize)
	return products, metadata, nil
}

func (p *ProductModel) buildFilters(builder sq.SelectBuilder, filters Filters) sq.SelectBuilder {
	if len(filters.IDs) > 0 {
		builder = builder.Where(sq.Eq{"ids": filters.IDs})
	}
	if filters.Name != "" {
		builder = builder.Where(
			"to_tsvector('simple', name) @@ plainto_tsquery('simple', ?)",
			filters.Name,
		)
	}
	if filters.DateFrom != nil {
		builder = builder.Where(sq.GtOrEq{"created_at": filters.DateFrom})
	}
	if filters.DateFrom != nil {
		builder = builder.Where(sq.LtOrEq{"created_at": filters.DateTo})
	}

	builder = builder.OrderBy(filters.sortColumns())
	builder = builder.Limit(uint64(filters.PageSize)).Offset(uint64(filters.offset()))
	return builder
}

func (p *ProductModel) countProducts(ctx context.Context, filters Filters) (int64, error) {
	builder := sq.Select("COUNT(*)").From("products")
	builder = p.buildFilters(builder, filters)

	query, args, _ := builder.ToSql()

	var totalRecords int64
	err := p.db.QueryRowContext(ctx, query, args...).Scan(&totalRecords)
	if err != nil {
		return 0, err
	}

	return totalRecords, nil
}

func (p *ProductModel) Update(ctx context.Context, product *Product) error {
	query, args, _ := sq.Update("products").
		Set("name", product.Name).
		Set("category_id", product.CategoryID).
		Set("description", product.Description).
		Set("price", product.Price).
		Set("quantity", product.Quantity).
		Set("version", product.Version+1).
		Where(sq.Eq{"id": product.ID}).
		Where(sq.Eq{"version": product.Version}).
		Suffix("RETURNING version").
		ToSql()

	err := p.db.QueryRowContext(ctx, query, args...).Scan(&product.Version)
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

func (p *ProductModel) Delete(ctx context.Context, id int64) error {
	query, _, _ := sq.Delete("products").Where(sq.Eq{"id": id}).ToSql()
	result, err := p.db.ExecContext(ctx, query, id)
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

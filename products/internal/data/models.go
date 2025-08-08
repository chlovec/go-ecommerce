package data

import (
	"errors"
	"fmt"
	"strings"
)

const ErrForeignKeyViolation = "23503"

var (
	ErrRecordNotFound    = errors.New("record not found")
	ErrEditConflict      = errors.New("edit conflict")
	ErrInvalidCategoryId = errors.New("invalid category_id")
)

type Models struct {
	Product  ProductRepository
	Category CategoryRepository
}

type Filters struct {
	Page     int
	PageSize int
	Sort     string
	// SortSafelist []string
}

// Check that the client-provided Sort field matches one of the entries in our safelist
// and if it does, extract the column name from the Sort field by stripping the leading
// hyphen character (if one exists).
func (f Filters) sortColumn() string {
	// for _, safeValue := range f.SortSafelist {
	// 	if f.Sort == safeValue {
	// 		return strings.TrimPrefix(f.Sort, "-")
	// 	}
	// }

	// panic("unsafe sort parameter: " + f.Sort)
	return strings.TrimPrefix(f.Sort, "-")
}

// Return the sort direction ("ASC" or "DESC") depending on the prefix character of the
// Sort field.
func (f Filters) sortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}

	return "ASC"
}

func (f Filters) sortColumnWithDirection() string {
	if f.Sort == "" {
		return ""
	}

	return fmt.Sprintf(" %s %s,", f.sortColumn(), f.sortDirection())
}

func (f Filters) limit() int {
	return f.PageSize
}

func (f Filters) offset() int {
	return (f.Page - 1) * f.PageSize
}

// Define a new Metadata struct for holding the pagination metadata.
type Metadata struct {
	CurrentPage  int `json:"current_page,omitzero"`
	PageSize     int `json:"page_size,omitzero"`
	FirstPage    int `json:"first_page,omitzero"`
	LastPage     int `json:"last_page,omitzero"`
	TotalRecords int `json:"total_records,omitzero"`
}

// The calculateMetadata() function calculates the appropriate pagination metadata
// values given the total number of records, current page, and page size values. Note
// that when the last page value is calculated we are dividing two int values, and
// when dividing integer types in Go the result will also be an integer type, with
// the modulus (or remainder) dropped. So, for example, if there were 12 records in total
// and a page size of 5, the last page value would be (12+5-1)/5 = 3.2, which is then
// truncated to 3 by Go.
func calculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		// Note that we return an empty Metadata struct if there are no records.
		return Metadata{}
	}

	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     (totalRecords + pageSize - 1) / pageSize,
		TotalRecords: totalRecords,
	}
}

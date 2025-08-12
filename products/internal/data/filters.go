package data

import (
	"strings"
	"time"
)

type Filters struct {
	IDs      []int64
	Name     string `validate:"omitempty,max=100"`
	DateFrom *time.Time
	DateTo   *time.Time
	Sorts    []string `validate:"omitempty,max=4,dive,oneof=id created_at name -id -created_at -name"`
	Page     int      `validate:"gte=1,lte=10_0000_000"`
	PageSize int      `validate:"gte=1,lte=100"`
}

func (f *Filters) sortColumns() string {
	sortFieldMapping := map[string]string{
		"created_at":  "created_at ASC",
		"-created_at": "created_at DESC",
		"id":          "id ASC",
		"-id":         "id DESC",
		"name":        "name ASC",
		"-name":       "name DESC",
	}

	hasId := false
	var sortColumns = []string{}
	for _, key := range f.Sorts {
		if key == "id" || key == "-id" {
			hasId = true
		}
		sortColumns = append(sortColumns, sortFieldMapping[key])
	}

	if !hasId {
		sortColumns = append(sortColumns, sortFieldMapping["id"])
	}

	return strings.Join(sortColumns, ", ")
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

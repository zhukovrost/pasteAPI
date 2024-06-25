package service

import (
	"errors"
	"math"
	"pasteAPI/internal/repository/models"
	"pasteAPI/pkg/validator"
	"strings"
)

type Filters struct {
	Page         uint32
	PageSize     uint32
	Sort         string
	SortSafelist []string
}

func ValidateFilters(v *validator.Validator, f Filters) {
	// Check that the page and page_size parameters contain sensible values.
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.Page <= 10_000_000, "page", "must be a maximum of 10 million")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must be a maximum of 100")
	// Check that the sort parameter matches a value in the safelist.
	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

func (f Filters) SortColumn() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}
	panic("unsafe sort parameter: " + f.Sort)
}

func (f Filters) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

func (f Filters) Limit() uint32 {
	return f.PageSize
}
func (f Filters) Offset() uint32 {
	return (f.Page - 1) * f.PageSize
}

// Metadata defines a new struct for holding the pagination metadata.
type Metadata struct {
	CurrentPage  uint32 `json:"current_page,omitempty"`
	PageSize     uint32 `json:"page_size,omitempty"`
	FirstPage    uint32 `json:"first_page,omitempty"`
	LastPage     uint32 `json:"last_page,omitempty"`
	TotalRecords uint32 `json:"total_records,omitempty"`
}

func CalculateMetadata(totalRecords, page, pageSize uint32) Metadata {
	if totalRecords == 0 {
		// Note that we return an empty Metadata struct if there are no records.
		return Metadata{}
	}
	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		FirstPage:    1,
		LastPage:     uint32(math.Ceil(float64(totalRecords) / float64(pageSize))),
		TotalRecords: totalRecords,
	}
}

type Categories []string

var CategoriesList Categories = []string{
	"Sport",
	"Home",
	"Work",
}

func (c Categories) GetCategory(categoryId uint8) (string, error) {
	if !c.IsValidCategory(categoryId) {
		return "", errors.New("category ID is out of range")
	}
	return c[categoryId-1], nil
}

func (c Categories) IsValidCategory(categoryId uint8) bool {
	if categoryId == 0 || int(categoryId) > len(c) {
		return false
	}
	return true
}

func ValidatePaste(v *validator.Validator, p *models.Paste) {
	v.Check(p.Title != "", "title", "must be provided")
	v.Check(len(p.Title) <= 255, "title", "must not be more than 500 bytes long")

	v.Check(CategoriesList.IsValidCategory(p.Category), "category", "no such category")

	v.Check(p.Text != "", "text", "must be provided")
	v.Check(len(p.Title) <= 500, "title", "must not be more than 500 bytes long")
}

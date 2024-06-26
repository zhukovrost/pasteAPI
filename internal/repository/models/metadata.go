package models

import "math"

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

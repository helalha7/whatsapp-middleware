package data

import "strings"

const maxPageSize = 100

type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafeList []string
}

func (f Filters) Validate() map[string]string {
	errors := make(map[string]string)

	if f.Page < 1 {
		errors["page"] = "must be greater than zero"
	}
	if f.PageSize < 1 {
		errors["page_size"] = "must be greater than zero"
	} else if f.PageSize > maxPageSize {
		errors["page_size"] = "must not be greater than 100"
	}
	if !f.sortIsSafe() {
		errors["sort"] = "invalid sort value"
	}

	if len(errors) == 0 {
		return nil
	}
	return errors
}

func (f Filters) Limit() int {
	return f.PageSize
}

func (f Filters) Offset() int {
	return (f.Page - 1) * f.PageSize
}

func (f Filters) SortColumn() string {
	if !f.sortIsSafe() {
		panic("unsafe sort parameter: " + f.Sort)
	}
	return strings.TrimPrefix(f.Sort, "-")
}

func (f Filters) SortDirection() string {
	if !f.sortIsSafe() {
		panic("unsafe sort parameter: " + f.Sort)
	}
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

func (f Filters) sortIsSafe() bool {
	for _, safeValue := range f.SortSafeList {
		if f.Sort == safeValue {
			return true
		}
	}
	return false
}

type Metadata struct {
	CurrentPage  int `json:"current_page,omitempty"`
	PageSize     int `json:"page_size,omitempty"`
	FirstPage    int `json:"first_page,omitempty"`
	LastPage     int `json:"last_page,omitempty"`
	TotalRecords int `json:"total_records,omitempty"`
}

func CalculateMetadata(totalRecords int, page int, pageSize int) Metadata {
	if totalRecords == 0 || page < 1 || pageSize < 1 {
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

package data

import "testing"

func TestFiltersValidate(t *testing.T) {
	tests := []struct {
		name       string
		filters    Filters
		wantFields []string
	}{
		{
			name: "valid filters",
			filters: Filters{
				Page:         1,
				PageSize:     20,
				Sort:         "name",
				SortSafeList: []string{"name", "-name"},
			},
		},
		{
			name: "invalid filters",
			filters: Filters{
				Page:         0,
				PageSize:     101,
				Sort:         "unsafe",
				SortSafeList: []string{"name", "-name"},
			},
			wantFields: []string{"page", "page_size", "sort"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.filters.Validate()
			if len(errors) != len(tt.wantFields) {
				t.Fatalf("expected %d errors, got %d: %v", len(tt.wantFields), len(errors), errors)
			}
			for _, field := range tt.wantFields {
				if errors[field] == "" {
					t.Errorf("expected validation error for %q", field)
				}
			}
		})
	}
}

func TestFiltersSQLValues(t *testing.T) {
	filters := Filters{
		Page:         3,
		PageSize:     25,
		Sort:         "-created_at",
		SortSafeList: []string{"created_at", "-created_at"},
	}

	if filters.Limit() != 25 {
		t.Errorf("expected limit 25, got %d", filters.Limit())
	}
	if filters.Offset() != 50 {
		t.Errorf("expected offset 50, got %d", filters.Offset())
	}
	if filters.SortColumn() != "created_at" {
		t.Errorf("expected sort column created_at, got %q", filters.SortColumn())
	}
	if filters.SortDirection() != "DESC" {
		t.Errorf("expected descending sort, got %q", filters.SortDirection())
	}
}

func TestFiltersSortColumnPanicsForUnsafeValue(t *testing.T) {
	filters := Filters{Sort: "unsafe", SortSafeList: []string{"name"}}
	defer func() {
		if recover() == nil {
			t.Fatal("expected unsafe sort value to panic")
		}
	}()

	filters.SortColumn()
}

func TestCalculateMetadata(t *testing.T) {
	metadata := CalculateMetadata(74, 2, 20)
	if metadata.CurrentPage != 2 || metadata.PageSize != 20 {
		t.Errorf("unexpected current page metadata: %+v", metadata)
	}
	if metadata.FirstPage != 1 || metadata.LastPage != 4 {
		t.Errorf("unexpected page range metadata: %+v", metadata)
	}
	if metadata.TotalRecords != 74 {
		t.Errorf("expected 74 total records, got %d", metadata.TotalRecords)
	}

	empty := CalculateMetadata(0, 1, 20)
	if empty != (Metadata{}) {
		t.Errorf("expected empty metadata, got %+v", empty)
	}
}

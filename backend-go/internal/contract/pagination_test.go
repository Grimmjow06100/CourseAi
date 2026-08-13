package contract

import "testing"

func TestSortDirectionValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		direction SortDirection
		wantError bool
	}{
		{name: "ascending", direction: SortAscending},
		{name: "descending", direction: SortDescending},
		{name: "invalid", direction: SortDirection("sideways"), wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.direction.Validate()
			if (err != nil) != test.wantError {
				t.Fatalf("Validate() error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}

func TestPaginationNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Pagination
		want Pagination
	}{
		{name: "defaults invalid values", in: Pagination{}, want: Pagination{Page: 1, PageSize: 20}},
		{name: "caps page size", in: Pagination{Page: 2, PageSize: 101}, want: Pagination{Page: 2, PageSize: 100}},
		{name: "keeps valid values", in: Pagination{Page: 3, PageSize: 25}, want: Pagination{Page: 3, PageSize: 25}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := test.in.Normalize(); got != test.want {
				t.Fatalf("Normalize() = %+v, want %+v", got, test.want)
			}
		})
	}
}

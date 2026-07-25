package contract

import "fmt"

type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

type Pagination struct {
	Page     int
	PageSize int
}

type Page[T any] struct {
	Items       []T
	Page        int
	PageSize    int
	TotalItems  int
	TotalPages  int
	HasNext     bool
	HasPrevious bool
}

func (d SortDirection) Validate() error {
	switch d {
	case SortAscending, SortDescending:
		return nil
	default:
		return fmt.Errorf("invalid sort direction: %s", d)
	}
}

func (p Pagination) Normalize() Pagination {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p
}

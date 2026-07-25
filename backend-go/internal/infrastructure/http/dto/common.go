package dto

import "time"

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type PageResponse[T any] struct {
	Items       []T  `json:"items"`
	Page        int  `json:"page"`
	PageSize    int  `json:"pageSize"`
	TotalItems  int  `json:"totalItems"`
	TotalPages  int  `json:"totalPages"`
	HasNext     bool `json:"hasNext"`
	HasPrevious bool `json:"hasPrevious"`
}

type TimestampResponse struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

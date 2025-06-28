package models

type WebResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type PaginationRequest struct {
	Page       int    `json:"page" validate:"omitempty,min=1"`
	Limit      int    `json:"limit" validate:"omitempty,min=1"`
	Order      string `json:"order" validate:"omitempty,oneof=asc desc"`
	OrderField string `json:"order_field" validate:"omitempty"`
}

type Pagination[T any] struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	TotalPages int  `json:"total_pages"`
	TotalItems int  `json:"total_items"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
	Items      T    `json:"items"`
}

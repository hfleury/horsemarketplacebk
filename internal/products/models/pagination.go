package models

type PaginatedProducts struct {
	Items []*Product `json:"items"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
	Limit int        `json:"limit"`
}

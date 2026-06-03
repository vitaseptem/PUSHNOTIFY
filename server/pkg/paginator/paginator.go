package paginator

import (
	"net/http"
	"strconv"
)

// Params holds normalized pagination parameters.
type Params struct {
	Page    int
	PerPage int
}

// Offset is the SQL OFFSET for the current page.
func (p Params) Offset() int { return (p.Page - 1) * p.PerPage }

// Limit is the SQL LIMIT for the current page.
func (p Params) Limit() int { return p.PerPage }

// FromRequest extracts page/per_page from the query string, clamped to sane
// bounds (page >= 1, 1 <= per_page <= 100, default 20).
func FromRequest(r *http.Request) Params {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return Params{Page: page, PerPage: perPage}
}

// Result wraps a paginated response.
type Result struct {
	Data    interface{} `json:"data"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
	Total   int         `json:"total"`
	Pages   int         `json:"pages"`
}

// NewResult builds a Result, computing the total page count.
func NewResult(data interface{}, p Params, total int) Result {
	pages := 0
	if p.PerPage > 0 {
		pages = (total + p.PerPage - 1) / p.PerPage
	}
	return Result{
		Data:    data,
		Page:    p.Page,
		PerPage: p.PerPage,
		Total:   total,
		Pages:   pages,
	}
}

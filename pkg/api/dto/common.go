package dto

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
	TotalCount int64 `json:"total_count"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Code    string                 `json:"code,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// ListQueryParams represents common list query parameters
type ListQueryParams struct {
	Page     int      `form:"page" validate:"min=1"`
	PageSize int      `form:"page_size" validate:"min=1,max=100"`
	SortBy   string   `form:"sort_by"`
	SortDesc bool     `form:"sort_desc"`
	State    string   `form:"state"`
	Tags     []string `form:"tags"`
}

// NewPaginationMeta creates a new PaginationMeta
func NewPaginationMeta(page, pageSize int, totalCount int64) PaginationMeta {
	totalPages := int(totalCount) / pageSize
	if int(totalCount)%pageSize > 0 {
		totalPages++
	}

	return PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		TotalCount: totalCount,
	}
}

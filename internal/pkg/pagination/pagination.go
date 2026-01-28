package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Errors
var (
	ErrInvalidCursor = errors.New("invalid cursor")
	ErrInvalidLimit  = errors.New("limit must be between 1 and max limit")
)

const (
	DefaultLimit = 10
	MaxLimit     = 100
)

// SortOrder represents sort direction
type SortOrder string

const (
	ASC  SortOrder = "ASC"
	DESC SortOrder = "DESC"
)

// Cursor-based Pagination
// Cursor represents a pagination cursor
type Cursor struct {
	ID        string    `json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
	SortValue any       `json:"sort_value,omitempty"`
}

// Encode encodes cursor to base64 string
func (c *Cursor) Encode() string {
	if c == nil {
		return ""
	}
	data, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes base64 string to Cursor
func DecodeCursor(encoded string) (*Cursor, error) {
	if encoded == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, ErrInvalidCursor
	}

	return &cursor, nil
}

// CursorRequest represents cursor-based pagination request
type CursorRequest struct {
	Cursor    string    `json:"cursor,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	SortField string    `json:"sort_field,omitempty"`
	SortOrder SortOrder `json:"sort_order,omitempty"`
}

// CursorResponse represents cursor-based pagination response
type CursorResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
	Total      int64  `json:"total,omitempty"`
}

// NewCursorRequest creates a new cursor request with defaults
func NewCursorRequest(cursor string, limit int) *CursorRequest {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	return &CursorRequest{
		Cursor:    cursor,
		Limit:     limit,
		SortOrder: DESC,
	}
}

// GetLimit returns validated limit
func (r *CursorRequest) GetLimit() int {
	if r.Limit <= 0 || r.Limit > MaxLimit {
		return DefaultLimit
	}
	return r.Limit
}

// GetFetchLimit returns limit+1 for checking hasMore
func (r *CursorRequest) GetFetchLimit() int {
	return r.GetLimit() + 1
}

// DecodedCursor returns the decoded cursor
func (r *CursorRequest) DecodedCursor() (*Cursor, error) {
	return DecodeCursor(r.Cursor)
}

// Offset-based Pagination
// OffsetRequest represents offset-based pagination request
type OffsetRequest struct {
	Page      int       `json:"page,omitempty"`
	PageSize  int       `json:"page_size,omitempty"`
	SortField string    `json:"sort_field,omitempty"`
	SortOrder SortOrder `json:"sort_order,omitempty"`
}

// OffsetResponse represents offset-based pagination response
type OffsetResponse[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// NewOffsetRequest creates a new offset request with defaults
func NewOffsetRequest(page, pageSize int) *OffsetRequest {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > MaxLimit {
		pageSize = DefaultLimit
	}
	return &OffsetRequest{
		Page:      page,
		PageSize:  pageSize,
		SortOrder: DESC,
	}
}

// GetOffset returns the offset for SQL query
func (r *OffsetRequest) GetOffset() int {
	return (r.GetPage() - 1) * r.GetPageSize()
}

// GetPage returns validated page
func (r *OffsetRequest) GetPage() int {
	if r.Page <= 0 {
		return 1
	}
	return r.Page
}

// GetPageSize returns validated page size
func (r *OffsetRequest) GetPageSize() int {
	if r.PageSize <= 0 || r.PageSize > MaxLimit {
		return DefaultLimit
	}
	return r.PageSize
}

// Response Builders
// BuildCursorResponse builds a cursor response from items
func BuildCursorResponse[T any](items []T, limit int, cursorBuilder func(T) *Cursor) *CursorResponse[T] {
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	resp := &CursorResponse[T]{
		Items:   items,
		HasMore: hasMore,
	}

	if len(items) > 0 && hasMore {
		lastItem := items[len(items)-1]
		cursor := cursorBuilder(lastItem)
		resp.NextCursor = cursor.Encode()
	}

	return resp
}

// BuildOffsetResponse builds an offset response from items and total count
func BuildOffsetResponse[T any](items []T, req *OffsetRequest, total int64) *OffsetResponse[T] {
	totalPages := int((total + int64(req.GetPageSize()) - 1) / int64(req.GetPageSize()))

	return &OffsetResponse[T]{
		Items:      items,
		Page:       req.GetPage(),
		PageSize:   req.GetPageSize(),
		TotalItems: total,
		TotalPages: totalPages,
		HasNext:    req.GetPage() < totalPages,
		HasPrev:    req.GetPage() > 1,
	}
}

// SQL Helpers
// SQLCursorCondition generates SQL WHERE condition for cursor pagination
func SQLCursorCondition(sortField string, order SortOrder) string {
	op := "<"
	if order == ASC {
		op = ">"
	}
	return fmt.Sprintf("(%s, id) %s ($1, $2)", sortField, op)
}

// SQLOrderBy generates ORDER BY clause
func SQLOrderBy(sortField string, order SortOrder) string {
	return fmt.Sprintf("%s %s, id %s", sortField, order, order)
}

// Elasticsearch Helpers
// ESSearchAfter returns search_after values for Elasticsearch
func ESSearchAfter(cursor *Cursor) []any {
	if cursor == nil {
		return nil
	}

	var values []any
	if !cursor.CreatedAt.IsZero() {
		values = append(values, cursor.CreatedAt.UnixMilli())
	}
	if cursor.SortValue != nil {
		values = append(values, cursor.SortValue)
	}
	if cursor.ID != "" {
		values = append(values, cursor.ID)
	}

	return values
}

// ESFromSize returns from and size for Elasticsearch offset pagination
func ESFromSize(req *OffsetRequest) (from, size int) {
	return req.GetOffset(), req.GetPageSize()
}

// ESSortFields returns sort fields for Elasticsearch
func ESSortFields(sortField string, order SortOrder) []map[string]any {
	esOrder := "desc"
	if order == ASC {
		esOrder = "asc"
	}

	return []map[string]any{
		{sortField: map[string]string{"order": esOrder}},
		{"_id": map[string]string{"order": esOrder}},
	}
}

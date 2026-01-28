package pagination

import (
	"testing"
	"time"
)

// Cursor Tests
func TestCursor_Encode(t *testing.T) {
	tests := []struct {
		name   string
		cursor *Cursor
		want   bool // true if should have output
	}{
		{
			name:   "nil cursor",
			cursor: nil,
			want:   false,
		},
		{
			name:   "cursor with ID",
			cursor: &Cursor{ID: "abc123"},
			want:   true,
		},
		{
			name:   "cursor with ID and CreatedAt",
			cursor: &Cursor{ID: "abc123", CreatedAt: time.Now()},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cursor.Encode()
			if tt.want && got == "" {
				t.Error("expected non-empty encoded cursor")
			}
			if !tt.want && got != "" {
				t.Error("expected empty encoded cursor")
			}
		})
	}
}

func TestDecodeCursor(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		wantErr error
		wantNil bool
	}{
		{
			name:    "empty string",
			encoded: "",
			wantErr: nil,
			wantNil: true,
		},
		{
			name:    "invalid base64",
			encoded: "!!!invalid!!!",
			wantErr: ErrInvalidCursor,
			wantNil: true,
		},
		{
			name:    "invalid json",
			encoded: "aW52YWxpZA==", // "invalid" in base64
			wantErr: ErrInvalidCursor,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeCursor(tt.encoded)
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("DecodeCursor() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantNil && got != nil {
				t.Errorf("DecodeCursor() = %v, want nil", got)
			}
		})
	}
}

func TestCursor_EncodeDecode_Roundtrip(t *testing.T) {
	original := &Cursor{
		ID:        "test-id-123",
		CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	encoded := original.Encode()
	if encoded == "" {
		t.Fatal("Encode() returned empty string")
	}

	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("DecodeCursor() error = %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %v, want %v", decoded.ID, original.ID)
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", decoded.CreatedAt, original.CreatedAt)
	}
}

// CursorRequest Tests
func TestNewCursorRequest(t *testing.T) {
	tests := []struct {
		name      string
		cursor    string
		limit     int
		wantLimit int
	}{
		{"default limit for zero", "", 0, DefaultLimit},
		{"default limit for negative", "", -1, DefaultLimit},
		{"max limit exceeded", "", 200, DefaultLimit},
		{"valid limit", "", 50, 50},
		{"with cursor", "abc", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewCursorRequest(tt.cursor, tt.limit)
			if req.GetLimit() != tt.wantLimit {
				t.Errorf("GetLimit() = %v, want %v", req.GetLimit(), tt.wantLimit)
			}
			if req.Cursor != tt.cursor {
				t.Errorf("Cursor = %v, want %v", req.Cursor, tt.cursor)
			}
		})
	}
}

func TestCursorRequest_GetFetchLimit(t *testing.T) {
	req := NewCursorRequest("", 20)
	if got := req.GetFetchLimit(); got != 21 {
		t.Errorf("GetFetchLimit() = %v, want 21", got)
	}
}

func TestCursorRequest_DecodedCursor(t *testing.T) {
	cursor := &Cursor{ID: "test"}
	encoded := cursor.Encode()

	req := NewCursorRequest(encoded, 10)
	decoded, err := req.DecodedCursor()
	if err != nil {
		t.Fatalf("DecodedCursor() error = %v", err)
	}
	if decoded.ID != "test" {
		t.Errorf("ID = %v, want test", decoded.ID)
	}
}

// OffsetRequest Tests

func TestNewOffsetRequest(t *testing.T) {
	tests := []struct {
		name         string
		page         int
		pageSize     int
		wantPage     int
		wantPageSize int
	}{
		{"defaults for zero", 0, 0, 1, DefaultLimit},
		{"defaults for negative", -1, -1, 1, DefaultLimit},
		{"max page size exceeded", 1, 200, 1, DefaultLimit},
		{"valid values", 5, 25, 5, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewOffsetRequest(tt.page, tt.pageSize)
			if req.GetPage() != tt.wantPage {
				t.Errorf("GetPage() = %v, want %v", req.GetPage(), tt.wantPage)
			}
			if req.GetPageSize() != tt.wantPageSize {
				t.Errorf("GetPageSize() = %v, want %v", req.GetPageSize(), tt.wantPageSize)
			}
		})
	}
}

func TestOffsetRequest_GetOffset(t *testing.T) {
	tests := []struct {
		page     int
		pageSize int
		want     int
	}{
		{1, 10, 0},
		{2, 10, 10},
		{3, 10, 20},
		{1, 20, 0},
		{5, 20, 80},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			req := NewOffsetRequest(tt.page, tt.pageSize)
			if got := req.GetOffset(); got != tt.want {
				t.Errorf("GetOffset() page=%d, pageSize=%d = %v, want %v",
					tt.page, tt.pageSize, got, tt.want)
			}
		})
	}
}

// Response Builder Tests

func TestBuildCursorResponse(t *testing.T) {
	t.Run("has more items", func(t *testing.T) {
		items := []string{"a", "b", "c", "d", "e", "f"}
		limit := 5

		resp := BuildCursorResponse(items, limit, func(item string) *Cursor {
			return &Cursor{ID: item}
		})

		if len(resp.Items) != 5 {
			t.Errorf("Items len = %d, want 5", len(resp.Items))
		}
		if !resp.HasMore {
			t.Error("HasMore = false, want true")
		}
		if resp.NextCursor == "" {
			t.Error("NextCursor is empty, want non-empty")
		}
	})

	t.Run("no more items", func(t *testing.T) {
		items := []string{"a", "b", "c"}
		limit := 5

		resp := BuildCursorResponse(items, limit, func(item string) *Cursor {
			return &Cursor{ID: item}
		})

		if len(resp.Items) != 3 {
			t.Errorf("Items len = %d, want 3", len(resp.Items))
		}
		if resp.HasMore {
			t.Error("HasMore = true, want false")
		}
		if resp.NextCursor != "" {
			t.Errorf("NextCursor = %s, want empty", resp.NextCursor)
		}
	})

	t.Run("empty items", func(t *testing.T) {
		items := []string{}
		limit := 5

		resp := BuildCursorResponse(items, limit, func(item string) *Cursor {
			return &Cursor{ID: item}
		})

		if len(resp.Items) != 0 {
			t.Errorf("Items len = %d, want 0", len(resp.Items))
		}
		if resp.HasMore {
			t.Error("HasMore = true, want false")
		}
	})
}

func TestBuildOffsetResponse(t *testing.T) {
	tests := []struct {
		name      string
		itemCount int
		page      int
		pageSize  int
		total     int64
		wantPages int
		wantNext  bool
		wantPrev  bool
	}{
		{"first page", 10, 1, 10, 25, 3, true, false},
		{"middle page", 10, 2, 10, 25, 3, true, true},
		{"last page", 5, 3, 10, 25, 3, false, true},
		{"single page", 5, 1, 10, 5, 1, false, false},
		{"empty", 0, 1, 10, 0, 0, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := make([]int, tt.itemCount)
			req := NewOffsetRequest(tt.page, tt.pageSize)
			resp := BuildOffsetResponse(items, req, tt.total)

			if resp.TotalPages != tt.wantPages {
				t.Errorf("TotalPages = %d, want %d", resp.TotalPages, tt.wantPages)
			}
			if resp.HasNext != tt.wantNext {
				t.Errorf("HasNext = %v, want %v", resp.HasNext, tt.wantNext)
			}
			if resp.HasPrev != tt.wantPrev {
				t.Errorf("HasPrev = %v, want %v", resp.HasPrev, tt.wantPrev)
			}
			if resp.Page != tt.page {
				t.Errorf("Page = %d, want %d", resp.Page, tt.page)
			}
		})
	}
}

// SQL Helper Tests

func TestSQLCursorCondition(t *testing.T) {
	tests := []struct {
		sortField string
		order     SortOrder
		want      string
	}{
		{"created_at", DESC, "(created_at, id) < ($1, $2)"},
		{"created_at", ASC, "(created_at, id) > ($1, $2)"},
		{"updated_at", DESC, "(updated_at, id) < ($1, $2)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.order), func(t *testing.T) {
			got := SQLCursorCondition(tt.sortField, tt.order)
			if got != tt.want {
				t.Errorf("SQLCursorCondition() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSQLOrderBy(t *testing.T) {
	tests := []struct {
		sortField string
		order     SortOrder
		want      string
	}{
		{"created_at", DESC, "created_at DESC, id DESC"},
		{"created_at", ASC, "created_at ASC, id ASC"},
		{"name", DESC, "name DESC, id DESC"},
	}

	for _, tt := range tests {
		t.Run(string(tt.order), func(t *testing.T) {
			got := SQLOrderBy(tt.sortField, tt.order)
			if got != tt.want {
				t.Errorf("SQLOrderBy() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Elasticsearch Helper Tests

func TestESSearchAfter(t *testing.T) {
	t.Run("nil cursor", func(t *testing.T) {
		got := ESSearchAfter(nil)
		if got != nil {
			t.Errorf("ESSearchAfter(nil) = %v, want nil", got)
		}
	})

	t.Run("cursor with CreatedAt and ID", func(t *testing.T) {
		cursor := &Cursor{
			ID:        "123",
			CreatedAt: time.Unix(1700000000, 0),
		}
		got := ESSearchAfter(cursor)
		if len(got) != 2 {
			t.Errorf("len(ESSearchAfter()) = %d, want 2", len(got))
		}
	})

	t.Run("cursor with only ID", func(t *testing.T) {
		cursor := &Cursor{ID: "123"}
		got := ESSearchAfter(cursor)
		if len(got) != 1 {
			t.Errorf("len(ESSearchAfter()) = %d, want 1", len(got))
		}
	})
}

func TestESFromSize(t *testing.T) {
	req := NewOffsetRequest(3, 20)
	from, size := ESFromSize(req)

	if from != 40 {
		t.Errorf("from = %d, want 40", from)
	}
	if size != 20 {
		t.Errorf("size = %d, want 20", size)
	}
}

func TestESSortFields(t *testing.T) {
	t.Run("DESC order", func(t *testing.T) {
		got := ESSortFields("created_at", DESC)
		if len(got) != 2 {
			t.Errorf("len(ESSortFields()) = %d, want 2", len(got))
		}
		if got[0]["created_at"].(map[string]string)["order"] != "desc" {
			t.Error("first sort field order should be desc")
		}
	})

	t.Run("ASC order", func(t *testing.T) {
		got := ESSortFields("name", ASC)
		if got[0]["name"].(map[string]string)["order"] != "asc" {
			t.Error("first sort field order should be asc")
		}
	})
}

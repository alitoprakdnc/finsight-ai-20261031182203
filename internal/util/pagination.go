package util

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// DefaultLimit is the number of items returned when the client does not specify a limit.
	DefaultLimit = 20
	// MaxLimit caps the maximum page size a client can request.
	MaxLimit = 100
)

// Pagination holds the parsed pagination parameters from a request.
type Pagination struct {
	// Limit is the maximum number of records to return (capped by MaxLimit).
	Limit int
	// Cursor is the decoded cursor value (usually the last seen ID).
	Cursor uint64
	// RawCursor is the original base64‑encoded cursor string supplied by the client.
	RawCursor string
	// Desc indicates whether results should be ordered descending (newest first).
	Desc bool
}

// ParsePagination extracts pagination query parameters from the Gin context.
// Supported query parameters:
//   - limit: integer (optional, defaults to DefaultLimit, max MaxLimit)
//   - cursor: base64‑encoded string representing the last seen ID (optional)
//   - order: "asc" or "desc" (optional, defaults to "desc")
func ParsePagination(c *gin.Context) (*Pagination, error) {
	// ----- limit -----
	limit := DefaultLimit
	if v := c.Query("limit"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid limit value: %w", err)
		}
		switch {
		case parsed <= 0:
			limit = DefaultLimit
		case parsed > MaxLimit:
			limit = MaxLimit
		default:
			limit = parsed
		}
	}

	// ----- cursor -----
	var cursor uint64
	rawCursor := c.Query("cursor")
	if rawCursor != "" {
		decoded, err := DecodeCursor(rawCursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor value: %w", err)
		}
		cursor = decoded
	}

	// ----- order -----
	order := strings.ToLower(c.DefaultQuery("order", "desc"))
	desc := true
	if order == "asc" {
		desc = false
	}

	return &Pagination{
		Limit:     limit,
		Cursor:    cursor,
		RawCursor: rawCursor,
		Desc:      desc,
	}, nil
}

// Paginate applies the pagination constraints to a GORM query.
// It orders by ID (asc/desc) and adds a cursor filter if supplied.
// The query limit is set to limit+1 so callers can detect the existence of a next page.
func Paginate(db *gorm.DB, p *Pagination) *gorm.DB {
	if p.Desc {
		db = db.Order("id DESC")
		if p.Cursor != 0 {
			db = db.Where("id < ?", p.Cursor)
		}
	} else {
		db = db.Order("id ASC")
		if p.Cursor != 0 {
			db = db.Where("id > ?", p.Cursor)
		}
	}
	// request one extra row to know whether there is a next page
	db = db.Limit(p.Limit + 1)
	return db
}

// BuildPageResponse trims the slice to the requested limit and, if there are more
// records, returns a cursor that points to the first record of the next page.
// The caller provides a getID function that extracts the uint64 identifier from an item.
func BuildPageResponse[T any](items []T, limit int, getID func(T) uint64) (result []T, nextCursor string) {
	if len(items) > limit {
		// There is at least one more record after the current page.
		nextID := getID(items[limit])
		nextCursor = EncodeCursor(nextID)
		result = items[:limit]
	} else {
		result = items
	}
	return result, nextCursor
}

// SetPaginationHeaders writes pagination‑related HTTP headers.
// Currently it sets X-Next-Cursor when a next page exists and disables caching.
func SetPaginationHeaders(c *gin.Context, nextCursor string) {
	if nextCursor != "" {
		c.Header("X-Next-Cursor", nextCursor)
	}
	// Prevent intermediate caches from serving stale paginated data.
	c.Header("Cache-Control", "no-store")
}

// EncodeCursor creates a URL‑safe base64 representation of a uint64 identifier.
func EncodeCursor(id uint64) string {
	s := strconv.FormatUint(id, 10)
	return base64.URLEncoding.EncodeToString([]byte(s))
}

// DecodeCursor reverses EncodeCursor, returning the original uint64 identifier.
func DecodeCursor(encoded string) (uint64, error) {
	b, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return 0, fmt.Errorf("base64 decode failed: %w", err)
	}
	id, err := strconv.ParseUint(string(b), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("cursor parse failed: %w", err)
	}
	return id, nil
}

// WritePaginatedResponse is a convenience helper that writes a JSON payload
// containing the data slice and pagination metadata.
// The payload shape matches typical REST conventions:
//
//	{
//	  "data": [...],
//	  "meta": {
//	    "nextCursor": "...",
//	    "limit": 20
//	  }
//	}
func WritePaginatedResponse[T any](c *gin.Context, items []T, limit int, getID func(T) uint64) {
	data, nextCursor := BuildPageResponse(items, limit, getID)

	// Populate response envelope
	resp := map[string]any{
		"data": data,
		"meta": map[string]any{
			"limit": limit,
		},
	}
	if nextCursor != "" {
		resp["meta"].(map[string]any)["nextCursor"] = nextCursor
	}

	// Set header for convenience (client may also read from body)
	SetPaginationHeaders(c, nextCursor)

	c.JSON(http.StatusOK, resp)
}
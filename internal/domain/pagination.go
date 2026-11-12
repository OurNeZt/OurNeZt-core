package domain

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

const DefaultPageSize = 50
const MaxPageSize = 100

// Page is a keyset window. A cursor is a position, never an authorization grant.
type Page struct {
	Size      int
	AfterTime *time.Time
	AfterID   string
}

type pageCursor struct {
	Version int       `json:"v"`
	Scope   string    `json:"s"`
	Time    time.Time `json:"t"`
	ID      string    `json:"i"`
}

func ParsePage(size int32, token, scope string) (Page, error) {
	page := Page{Size: int(size)}
	invalid := fmt.Errorf("%w: invalid page size or page token", apperror.ErrInvalidArgument)
	if page.Size < 0 || page.Size > MaxPageSize || len(token) > 512 {
		return Page{}, invalid
	}
	if page.Size == 0 {
		page.Size = DefaultPageSize
	}
	if token == "" {
		return page, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return Page{}, invalid
	}
	var cursor pageCursor
	if json.Unmarshal(raw, &cursor) != nil || cursor.Version != 1 || cursor.Scope != scope || cursor.Time.IsZero() || !validCursorID(cursor.ID) {
		return Page{}, invalid
	}
	page.AfterTime, page.AfterID = &cursor.Time, cursor.ID
	return page, nil
}

func validCursorID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, ch := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if ch != '-' {
				return false
			}
			continue
		}
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func PageResult[T any](rows []T, page Page, scope string, key func(T) (time.Time, ID)) ([]T, string) {
	if len(rows) <= page.Size {
		return rows, ""
	}
	rows = rows[:page.Size]
	stamp, id := key(rows[len(rows)-1])
	raw, _ := json.Marshal(pageCursor{Version: 1, Scope: scope, Time: stamp, ID: string(id)})
	return rows, base64.RawURLEncoding.EncodeToString(raw)
}

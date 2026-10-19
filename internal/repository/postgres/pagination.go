package postgres

import (
	"fmt"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

// Internal aggregate reads must be complete or fail, never silently truncate.
// Public lists provide a page explicitly and fetch one extra row for the cursor.
func listWindow(pages []domain.Page) (int, *time.Time, string, error) {
	if len(pages) == 0 {
		return domain.MaxPageSize + 1, nil, "", nil
	}
	page := pages[0]
	if len(pages) != 1 || page.Size < 1 || page.Size > domain.MaxPageSize {
		return 0, nil, "", apperror.ErrInvalidArgument
	}
	return page.Size + 1, page.AfterTime, page.AfterID, nil
}

func completeList[T any](rows []T, pages []domain.Page) ([]T, error) {
	if len(pages) == 0 && len(rows) > domain.MaxPageSize {
		return nil, fmt.Errorf("%w: aggregate exceeds %d records; use pagination", apperror.ErrQuotaExceeded, domain.MaxPageSize)
	}
	return rows, nil
}

package domain

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

func TestResourcePaginationCursor(t *testing.T) {
	stamp := time.Date(2026, 10, 19, 1, 2, 3, 456789000, time.UTC)
	rows := []User{{ID: "00000000-0000-0000-0000-000000000001", CreatedAt: stamp}, {ID: "00000000-0000-0000-0000-000000000002", CreatedAt: stamp}}
	page, err := ParsePage(1, "", "users")
	if err != nil {
		t.Fatal(err)
	}
	got, token := PageResult(rows, page, "users", func(u User) (time.Time, ID) { return u.CreatedAt, u.ID })
	if len(got) != 1 || token == "" {
		t.Fatal("missing bounded page or continuation")
	}
	next, err := ParsePage(1, token, "users")
	if err != nil || next.AfterID != string(rows[0].ID) || !next.AfterTime.Equal(stamp) {
		t.Fatalf("cursor lost precision or position: %+v %v", next, err)
	}
	if _, err := ParsePage(1, token, "another-user"); !errors.Is(err, apperror.ErrInvalidArgument) {
		t.Fatal("cursor reused across scopes")
	}
	if _, token := PageResult(rows[:1], page, "users", func(u User) (time.Time, ID) { return u.CreatedAt, u.ID }); token != "" {
		t.Fatal("terminal page has a continuation")
	}
	defaultPage, err := ParsePage(0, "", "users")
	if err != nil || defaultPage.Size != DefaultPageSize {
		t.Fatal("default page size")
	}
	for _, size := range []int32{-1, MaxPageSize + 1} {
		if _, err := ParsePage(size, "", "users"); !errors.Is(err, apperror.ErrInvalidArgument) {
			t.Fatal("invalid page size accepted")
		}
	}
	for _, token := range []string{"not-a-token", strings.Repeat("x", 513), "e30"} {
		if _, err := ParsePage(1, token, "users"); !errors.Is(err, apperror.ErrInvalidArgument) {
			t.Fatal("invalid token accepted")
		}
	}
}

func TestResourceCredentialInputBoundaries(t *testing.T) {
	if err := ValidateAuthInputLengths(strings.Repeat("a", 254), strings.Repeat("界", 120), strings.Repeat("x", 1024)); err != nil {
		t.Fatal(err)
	}
	for _, input := range [][3]string{{strings.Repeat("a", 255), "", ""}, {"", strings.Repeat("界", 121), ""}, {"", "", strings.Repeat("x", 1025)}, {"", "", strings.Repeat("界", 342)}} {
		if err := ValidateAuthInputLengths(input[0], input[1], input[2]); !errors.Is(err, apperror.ErrInvalidArgument) {
			t.Fatal("oversized auth input accepted")
		}
	}
}

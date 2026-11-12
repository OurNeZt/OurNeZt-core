package server

import (
	"context"
	"strings"
	"testing"

	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestResourceInputLimitsBeforeHandler(t *testing.T) {
	for _, tc := range []struct {
		name    string
		req     proto.Message
		allowed bool
	}{
		{"email boundary", &pb.LoginRequest{Email: strings.Repeat("a", 254)}, true},
		{"email too long", &pb.LoginRequest{Email: strings.Repeat("a", 255)}, false},
		{"password boundary", &pb.LoginRequest{Password: strings.Repeat("x", 1024)}, true},
		{"password bytes", &pb.LoginRequest{Password: strings.Repeat("界", 342)}, false},
		{"unicode name boundary", &pb.CreateFamilyRequest{Name: strings.Repeat("界", 120)}, true},
		{"unicode name too long", &pb.CreateFamilyRequest{Name: strings.Repeat("界", 121)}, false},
		{"housing notes boundary", &pb.UpdateHousingNotesRequest{Notes: strings.Repeat("界", 10000)}, true},
		{"housing notes too long", &pb.UpdateHousingNotesRequest{Notes: strings.Repeat("界", 10001)}, false},
		{"answer notes too long", &pb.HousingAnswer{Notes: strings.Repeat("x", 5001)}, false},
		{"location too long", &pb.HousingOption{Location: strings.Repeat("x", 256)}, false},
		{"unit type too long", &pb.HousingOption{UnitType: strings.Repeat("x", 121)}, false},
		{"oversized reorder", &pb.ReorderHousingCriteriaRequest{CriterionIds: make([]string, 51)}, false},
		{"oversized overrides", &pb.HousingOption{DiaIncomeOverrides: make([]*pb.HousingDIAIncomeOverride, 51)}, false},
		{"nested oversized input", &pb.CalculateHousingAffordabilityRequest{HousingOption: &pb.HousingOption{Name: strings.Repeat("x", 121)}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			_, err := InputLimitInterceptor(context.Background(), tc.req, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) { called = true; return nil, nil })
			if called != tc.allowed || tc.allowed && err != nil || !tc.allowed && status.Code(err) != codes.InvalidArgument {
				t.Fatalf("called=%v err=%v", called, err)
			}
		})
	}
}

func TestResourceQuotaStatus(t *testing.T) {
	if status.Code(toStatusError(apperror.ErrQuotaExceeded)) != codes.ResourceExhausted {
		t.Fatal("quota error not exposed clearly")
	}
}

func TestResourcePaginationRejectsInvalidBeforeLookup(t *testing.T) {
	s := NewFamilyServer(&fakeFamiliesRepository{}, 0, nil)
	for _, req := range []*pb.ListUserFamiliesRequest{{UserId: "user", PageSize: -1}, {UserId: "user", PageSize: 101}, {UserId: "user", PageToken: "bad-token"}} {
		if _, err := s.ListUserFamilies(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("invalid pagination accepted: %v", err)
		}
	}
}

package server

import (
	"context"
	"strings"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (r *fakeHousingRepository) UpdateHousingNotes(_ context.Context, _ domain.ID, notes string, _ domain.ID) (string, error) {
	return notes, nil
}

type notesRepository struct {
	fakeHousingRepository
	actor, housingID domain.ID
	calls            int
	err              error
}

func (r *notesRepository) UpdateHousingNotes(_ context.Context, id domain.ID, notes string, actor domain.ID) (string, error) {
	r.actor, r.housingID = actor, id
	r.calls++
	return notes, r.err
}

func TestHousingNotesAuthenticationAndValidation(t *testing.T) {
	ctx := context.Background()
	repo := &notesRepository{}
	s := NewHousingServer(repo, nil)
	if _, err := s.UpdateHousingNotes(ctx, &pb.UpdateHousingNotesRequest{HousingId: "home"}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("unauthenticated save: %v", err)
	}
	s.auth = checklistTestAuth{id: "session-user"}
	for _, req := range []*pb.UpdateHousingNotesRequest{nil, {}, {HousingId: "home", Notes: strings.Repeat("a", 10001)}, {HousingId: "home", Notes: string([]byte{0xff})}} {
		if _, err := s.UpdateHousingNotes(ctx, req); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("invalid request: %v", err)
		}
	}
	if repo.calls != 0 {
		t.Fatal("invalid request reached repository")
	}
	for _, notes := range []string{"", "  Viewing feedback\nAsk agent about renovation.\n", strings.Repeat("🏡", 10000)} {
		got, err := s.UpdateHousingNotes(ctx, &pb.UpdateHousingNotesRequest{HousingId: "home", Notes: notes})
		if err != nil || got.GetNotes() != notes || repo.actor != "session-user" || repo.housingID != "home" {
			t.Fatalf("notes or authenticated actor lost: %v", err)
		}
	}
	repo.err = apperror.ErrForbidden
	if _, err := s.UpdateHousingNotes(ctx, &pb.UpdateHousingNotesRequest{HousingId: "home"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("authorization error lost: %v", err)
	}
}

func TestHousingNotesCreateAndRead(t *testing.T) {
	notes := "  Viewing feedback\nNear MRT 🏡\n"
	repo := &fakeHousingRepository{created: domain.HousingOption{Notes: notes}, got: domain.HousingOption{Notes: notes}, list: []domain.HousingOption{{Notes: notes}}}
	s := NewHousingServer(repo, &fakePeopleRepository{})
	input := &pb.HousingOption{FamilyId: "family", Name: "Home", HousingType: "bto", LoanType: "cash", Notes: notes}
	created, err := s.CreateHousingOption(context.Background(), input)
	if err != nil || created.GetNotes() != notes || repo.createInput.Notes != notes {
		t.Fatalf("create notes: %v", err)
	}
	got, err := s.GetHousingOption(context.Background(), &pb.GetHousingOptionRequest{HousingId: "home", ViewerUserId: "viewer"})
	if err != nil || got.GetNotes() != notes {
		t.Fatalf("get notes: %v", err)
	}
	list, err := s.ListHousingOptions(context.Background(), &pb.ListHousingOptionsRequest{FamilyId: "family", ViewerUserId: "viewer"})
	if err != nil || len(list.GetHousingOptions()) != 1 || list.HousingOptions[0].GetNotes() != notes {
		t.Fatalf("list notes: %v", err)
	}
	input.Notes = strings.Repeat("a", 10001)
	if _, err := s.CreateHousingOption(context.Background(), input); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("oversized create: %v", err)
	}
}

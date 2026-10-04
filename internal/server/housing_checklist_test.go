package server

import (
	"context"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

type checklistTestAuth struct{ id domain.ID }

func (a checklistTestAuth) Authenticate(context.Context) (domain.User, error) {
	return domain.User{ID: a.id}, nil
}

type checklistTestRepo struct {
	repository.HousingChecklist
	actor     domain.ID
	saveCalls int
	err       error
}

func (r *checklistTestRepo) SaveHousingAnswer(_ context.Context, a domain.HousingAnswer, actor domain.ID) (domain.HousingAnswer, error) {
	r.actor = actor
	r.saveCalls++
	return a, r.err
}
func (r *checklistTestRepo) ListHousingCriteria(_ context.Context, _, actor domain.ID) ([]domain.HousingCriterion, error) {
	r.actor = actor
	return []domain.HousingCriterion{{ID: "c", FamilyID: "f", Name: "Location"}}, r.err
}
func (r *checklistTestRepo) ListHousingAnswers(_ context.Context, _, actor domain.ID) ([]domain.HousingAnswer, error) {
	r.actor = actor
	rating := int32(4)
	return []domain.HousingAnswer{{HousingID: "h", CriterionID: "c", State: "complete", Rating: &rating}, {HousingID: "other", CriterionID: "c", State: "pending"}, {HousingID: "h", CriterionID: "retired", State: "complete", Rating: &rating}}, r.err
}

func TestChecklistRequiresAuthentication(t *testing.T) {
	s := HousingServer{checklist: &checklistTestRepo{}}
	ctx := context.Background()
	calls := []func() error{
		func() error {
			_, e := s.ListHousingCriteria(ctx, &pb.ListHousingCriteriaRequest{FamilyId: "f"})
			return e
		},
		func() error { _, e := s.SaveHousingCriterion(ctx, &pb.HousingCriterion{}); return e },
		func() error { _, e := s.DeleteHousingCriterion(ctx, &pb.DeleteHousingCriterionRequest{}); return e },
		func() error { _, e := s.ReorderHousingCriteria(ctx, &pb.ReorderHousingCriteriaRequest{}); return e },
		func() error {
			_, e := s.GetHousingEvaluation(ctx, &pb.GetHousingOptionRequest{ViewerUserId: "spoofed", HousingId: "h"})
			return e
		},
		func() error { _, e := s.SaveHousingAnswer(ctx, &pb.HousingAnswer{}); return e },
	}
	for i, call := range calls {
		if status.Code(call()) != codes.Unauthenticated {
			t.Errorf("call %d allowed unauthenticated request", i)
		}
	}
}
func TestChecklistUsesSessionAndValidatesBeforeWrite(t *testing.T) {
	repo := &checklistTestRepo{}
	s := HousingServer{checklist: repo, auth: checklistTestAuth{id: "session-user"}, housing: &fakeHousingRepository{got: domain.HousingOption{ID: "h", FamilyID: "f"}}}
	ctx := context.Background()
	_, err := s.SaveHousingAnswer(ctx, &pb.HousingAnswer{HousingId: "h", CriterionId: "c", State: "complete"})
	if status.Code(err) != codes.InvalidArgument || repo.saveCalls != 0 {
		t.Fatal("invalid answer reached repository")
	}
	_, err = s.SaveHousingAnswer(ctx, &pb.HousingAnswer{HousingId: "h", CriterionId: "c", State: "pending", Notes: "Needs checking"})
	if err != nil || repo.actor != "session-user" || repo.saveCalls != 1 {
		t.Fatalf("session actor not used: %v %+v", err, repo)
	}
	eval, err := s.GetHousingEvaluation(ctx, &pb.GetHousingOptionRequest{HousingId: "h", ViewerUserId: "spoofed"})
	if err != nil || len(eval.GetAnswers()) != 1 || eval.GetSummary().GetScore() != 4 || repo.actor != "session-user" {
		t.Fatalf("incorrect evaluation: %v %v", eval, err)
	}
	repo.err = apperror.ErrForbidden
	_, err = s.ListHousingCriteria(ctx, &pb.ListHousingCriteriaRequest{FamilyId: "foreign"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("authorization error lost: %v", err)
	}
	_, err = s.ReorderHousingCriteria(ctx, &pb.ReorderHousingCriteriaRequest{FamilyId: "f", CriterionIds: []string{"c", "c"}})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("duplicate ordering accepted: %v", err)
	}
}

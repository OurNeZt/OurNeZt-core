package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"github.com/OurNeZt/ournezt-core/internal/platform/security"
	"github.com/OurNeZt/ournezt-core/internal/server"
	"github.com/OurNeZt/ournezt-core/internal/service"
	"google.golang.org/grpc/metadata"
)

func TestResourceLimitsPostgresChecklistAtHousingCapacity(t *testing.T) {
	f := familyInviteFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resourceExec(t, f, `INSERT INTO housing_options(family_id,name,housing_type,purchase_price_cents,loan_type)
		SELECT $1,'Home '||i,CASE i%3 WHEN 0 THEN 'bto' WHEN 1 THEN 'resale_hdb' ELSE 'private_condo' END,1,'cash'
		FROM generate_series(1,100) i`, f.family.ID)
	resourceExec(t, f, `INSERT INTO housing_checklist_criteria(family_id,name,display_order,weight)
		SELECT $1,'Criterion '||i,i-1,2 FROM generate_series(1,50) i`, f.family.ID)
	resourceExec(t, f, `INSERT INTO housing_checklist_answers(housing_id,criterion_id,family_id,state,rating,notes)
		SELECT h.id,c.id,h.family_id,'complete',((substring(h.name from 6)::int-1)%5)+1,h.name||' / '||c.name
		FROM housing_options h JOIN housing_checklist_criteria c ON c.family_id=h.family_id WHERE h.family_id=$1`, f.family.ID)
	repo := NewHousingRepository(f.pool)
	answers, err := repo.ListHousingAnswers(ctx, f.family.ID, f.owner.ID)
	if err != nil || len(answers) != 5000 {
		t.Fatalf("full checklist matrix: answers=%d err=%v", len(answers), err)
	}
	options, err := repo.ListHousingOptions(ctx, f.family.ID, f.owner.ID)
	if err != nil || len(options) != 100 {
		t.Fatalf("complete aggregate: options=%d err=%v", len(options), err)
	}
	expected := make(map[string]float64, len(options))
	for _, option := range options {
		var number int
		if _, err := fmt.Sscanf(option.Name, "Home %d", &number); err != nil {
			t.Fatal(err)
		}
		score := float64((number-1)%5 + 1)
		expected[string(option.ID)] = score
		summary := option.Evaluation
		if summary == nil || summary.Score == nil || *summary.Score != score || summary.Total != 50 || summary.Completed != 50 || summary.Rated != 50 || !summary.Weighted {
			t.Fatalf("incorrect aggregate summary for %s: %+v", option.Name, summary)
		}
	}
	users := NewUserRepository(f.pool)
	if err := users.CreateSession(ctx, f.owner.ID, f.owner.PasswordHash, security.HashToken("capacity-token"), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	auth := server.NewAuthServer(service.NewAuthService(users, security.DefaultArgon2Params()), users, 32, time.Hour, nil)
	housing := server.NewHousingServer(repo, NewPersonRepository(f.pool), auth)
	ctx = metadata.NewIncomingContext(ctx, metadata.Pairs("x-session-token", "capacity-token"))
	first, err := housing.ListHousingOptions(ctx, &pb.ListHousingOptionsRequest{FamilyId: string(f.family.ID)})
	if err != nil || len(first.GetHousingOptions()) != 50 || first.GetNextPageToken() == "" {
		t.Fatalf("first default page: %v", err)
	}
	second, err := housing.ListHousingOptions(ctx, &pb.ListHousingOptionsRequest{FamilyId: string(f.family.ID), PageToken: first.NextPageToken})
	if err != nil || len(second.GetHousingOptions()) != 50 || second.GetNextPageToken() != "" {
		t.Fatalf("second default page: %v", err)
	}
	all, err := housing.ListHousingOptions(ctx, &pb.ListHousingOptionsRequest{FamilyId: string(f.family.ID), PageSize: 100})
	if err != nil || len(all.GetHousingOptions()) != 100 || all.GetNextPageToken() != "" {
		t.Fatalf("maximum-size page: %v", err)
	}
	for _, collection := range [][]*pb.HousingOption{append(first.HousingOptions, second.HousingOptions...), all.HousingOptions} {
		seen := make(map[string]bool)
		for _, option := range collection {
			score, exists := expected[option.Id]
			summary := option.GetEvaluation()
			if !exists || seen[option.Id] || summary.GetScore() != score || summary.GetTotal() != 50 || summary.GetCompleted() != 50 || summary.GetRated() != 50 || !summary.GetWeighted() {
				t.Fatalf("missing/duplicate option or incorrect page score: %v", option)
			}
			seen[option.Id] = true
		}
		if len(seen) != 100 {
			t.Fatalf("pagination returned %d unique options, want 100", len(seen))
		}
	}
	// The last option remains editable and its detailed evaluation includes only
	// its own 50 answers, even while the family has all 5,000 answers populated.
	target := second.HousingOptions[len(second.HousingOptions)-1]
	evaluation, err := housing.GetHousingEvaluation(ctx, &pb.GetHousingOptionRequest{HousingId: target.Id})
	if err != nil || len(evaluation.GetCriteria()) != 50 || len(evaluation.GetAnswers()) != 50 || evaluation.GetSummary().GetScore() != expected[target.Id] {
		t.Fatalf("single-option evaluation at capacity: %v", err)
	}
	for _, answer := range evaluation.Answers {
		if answer.HousingId != target.Id || !strings.HasPrefix(answer.Notes, target.Name+" / ") {
			t.Fatal("another option's answers or notes leaked into the evaluation")
		}
	}
	summaries, err := repo.listHousingAnswers(ctx, f.family.ID, f.owner.ID, false, []domain.ID{domain.ID(target.Id)})
	if err != nil || len(summaries) != 50 {
		t.Fatalf("selected-option summary answers: count=%d err=%v", len(summaries), err)
	}
	for _, answer := range summaries {
		if answer.HousingID != domain.ID(target.Id) || answer.Notes != "" {
			t.Fatal("list summary loaded unrelated answers or detailed notes")
		}
	}
	updated := &pb.HousingAnswer{HousingId: target.Id, CriterionId: evaluation.Criteria[0].Id, State: "not_applicable", Notes: "Saved at capacity"}
	if _, err := housing.SaveHousingAnswer(ctx, updated); err != nil {
		t.Fatalf("answer edit at capacity: %v", err)
	}
	evaluation, err = housing.GetHousingEvaluation(ctx, &pb.GetHousingOptionRequest{HousingId: target.Id})
	if err != nil || evaluation.GetSummary().GetCompleted() != 49 || evaluation.GetSummary().GetNotApplicable() != 1 || evaluation.GetSummary().GetRated() != 49 || evaluation.GetSummary().GetScore() != expected[target.Id] {
		t.Fatalf("edited evaluation: %v %v", evaluation, err)
	}
	found := false
	for _, answer := range evaluation.GetAnswers() {
		if answer.CriterionId == updated.CriterionId {
			found = answer.State == "not_applicable" && answer.Rating == nil && answer.Notes == updated.Notes
		}
	}
	if !found {
		t.Fatal("answer edit was not persisted")
	}
	if _, err := repo.SaveHousingCriterion(ctx, domain.HousingCriterion{FamilyID: f.family.ID, Name: "Over quota", DisplayOrder: 50}, f.owner.ID); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatalf("50-criterion limit changed: %v", err)
	}
	if _, err := repo.CreateHousingOption(ctx, domain.HousingOption{FamilyID: f.family.ID, Name: "Over quota", Type: domain.HousingTypeBTO, LoanType: domain.LoanTypeCash}, f.owner.ID); !errors.Is(err, apperror.ErrQuotaExceeded) {
		t.Fatalf("101st housing option accepted: %v", err)
	}
}

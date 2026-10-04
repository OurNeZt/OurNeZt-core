package server

import (
	"context"
	"strings"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	pb "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s HousingServer) checklistActor(ctx context.Context) (domain.ID, error) {
	id, err := authenticateUserID(ctx, s.auth)
	if err != nil {
		return "", toStatusError(err)
	}
	if s.checklist == nil {
		return "", status.Error(codes.Unimplemented, "housing checklist repository unavailable")
	}
	return id, nil
}

func criterionToProto(c domain.HousingCriterion) *pb.HousingCriterion {
	return &pb.HousingCriterion{Id: string(c.ID), FamilyId: string(c.FamilyID), Name: c.Name, Description: c.Description, DisplayOrder: c.DisplayOrder, Weight: c.Weight}
}
func answerToProto(a domain.HousingAnswer) *pb.HousingAnswer {
	return &pb.HousingAnswer{HousingId: string(a.HousingID), CriterionId: string(a.CriterionID), State: a.State, Rating: a.Rating, Notes: a.Notes}
}
func evaluationSummaryToProto(s *domain.HousingEvaluationSummary) *pb.HousingEvaluationSummary {
	if s == nil {
		return nil
	}
	return &pb.HousingEvaluationSummary{Score: s.Score, Total: s.Total, Completed: s.Completed, NotApplicable: s.NotApplicable, Rated: s.Rated, Weighted: s.Weighted}
}

func (s HousingServer) ListHousingCriteria(ctx context.Context, req *pb.ListHousingCriteriaRequest) (*pb.ListHousingCriteriaResponse, error) {
	actor, err := s.checklistActor(ctx)
	if err != nil {
		return nil, err
	}
	familyID, err := requireID(req.GetFamilyId())
	if err != nil {
		return nil, toStatusError(err)
	}
	criteria, err := s.checklist.ListHousingCriteria(ctx, familyID, actor)
	if err != nil {
		return nil, toStatusError(err)
	}
	resp := &pb.ListHousingCriteriaResponse{}
	for _, c := range criteria {
		resp.Criteria = append(resp.Criteria, criterionToProto(c))
	}
	return resp, nil
}

func (s HousingServer) SaveHousingCriterion(ctx context.Context, req *pb.HousingCriterion) (*pb.HousingCriterion, error) {
	actor, err := s.checklistActor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	c := domain.HousingCriterion{ID: domain.ID(strings.TrimSpace(req.Id)), FamilyID: domain.ID(strings.TrimSpace(req.FamilyId)), Name: strings.TrimSpace(req.Name), Description: strings.TrimSpace(req.Description), DisplayOrder: req.DisplayOrder, Weight: req.Weight}
	if err := domain.ValidateHousingCriterion(c); err != nil {
		return nil, toStatusError(err)
	}
	saved, err := s.checklist.SaveHousingCriterion(ctx, c, actor)
	if err != nil {
		return nil, toStatusError(err)
	}
	return criterionToProto(saved), nil
}

func (s HousingServer) DeleteHousingCriterion(ctx context.Context, req *pb.DeleteHousingCriterionRequest) (*pb.DeleteHousingCriterionResponse, error) {
	actor, err := s.checklistActor(ctx)
	if err != nil {
		return nil, err
	}
	familyID, err := requireID(req.GetFamilyId())
	if err != nil {
		return nil, toStatusError(err)
	}
	id, err := requireID(req.GetCriterionId())
	if err != nil {
		return nil, toStatusError(err)
	}
	if err := s.checklist.DeleteHousingCriterion(ctx, familyID, id, actor); err != nil {
		return nil, toStatusError(err)
	}
	return &pb.DeleteHousingCriterionResponse{}, nil
}

func (s HousingServer) ReorderHousingCriteria(ctx context.Context, req *pb.ReorderHousingCriteriaRequest) (*pb.ListHousingCriteriaResponse, error) {
	actor, err := s.checklistActor(ctx)
	if err != nil {
		return nil, err
	}
	familyID, err := requireID(req.GetFamilyId())
	if err != nil {
		return nil, toStatusError(err)
	}
	ids := make([]domain.ID, 0, len(req.GetCriterionIds()))
	seen := map[domain.ID]bool{}
	for _, raw := range req.GetCriterionIds() {
		id, err := requireID(raw)
		if err != nil || seen[id] {
			return nil, toStatusError(apperror.ErrInvalidArgument)
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if err := s.checklist.ReorderHousingCriteria(ctx, familyID, ids, actor); err != nil {
		return nil, toStatusError(err)
	}
	return s.ListHousingCriteria(ctx, &pb.ListHousingCriteriaRequest{FamilyId: string(familyID)})
}

func (s HousingServer) GetHousingEvaluation(ctx context.Context, req *pb.GetHousingOptionRequest) (*pb.HousingEvaluation, error) {
	actor, err := s.checklistActor(ctx)
	if err != nil {
		return nil, err
	}
	id, err := requireID(req.GetHousingId())
	if err != nil {
		return nil, toStatusError(err)
	}
	option, err := s.housing.GetHousingOption(ctx, id, actor)
	if err != nil {
		return nil, toStatusError(err)
	}
	criteria, err := s.checklist.ListHousingCriteria(ctx, option.FamilyID, actor)
	if err != nil {
		return nil, toStatusError(err)
	}
	answers, err := s.checklist.ListHousingAnswers(ctx, option.FamilyID, actor)
	if err != nil {
		return nil, toStatusError(err)
	}
	resp := &pb.HousingEvaluation{}
	active := map[domain.ID]bool{}
	for _, c := range criteria {
		resp.Criteria = append(resp.Criteria, criterionToProto(c))
		active[c.ID] = true
	}
	selected := []domain.HousingAnswer{}
	for _, a := range answers {
		if a.HousingID == id && active[a.CriterionID] {
			selected = append(selected, a)
			resp.Answers = append(resp.Answers, answerToProto(a))
		}
	}
	summary := domain.SummarizeHousingEvaluation(criteria, selected)
	resp.Summary = evaluationSummaryToProto(&summary)
	return resp, nil
}

func (s HousingServer) SaveHousingAnswer(ctx context.Context, req *pb.HousingAnswer) (*pb.HousingAnswer, error) {
	actor, err := s.checklistActor(ctx)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, toStatusError(apperror.ErrInvalidArgument)
	}
	a := domain.HousingAnswer{HousingID: domain.ID(strings.TrimSpace(req.HousingId)), CriterionID: domain.ID(strings.TrimSpace(req.CriterionId)), State: req.State, Rating: req.Rating, Notes: strings.TrimSpace(req.Notes)}
	if err := domain.ValidateHousingAnswer(a); err != nil {
		return nil, toStatusError(err)
	}
	saved, err := s.checklist.SaveHousingAnswer(ctx, a, actor)
	if err != nil {
		return nil, toStatusError(err)
	}
	return answerToProto(saved), nil
}

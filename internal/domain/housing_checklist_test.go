package domain

import (
	"math"
	"testing"
)

func checklistPtr[T any](v T) *T { return &v }

func TestHousingEvaluationScoreAndProgress(t *testing.T) {
	criteria := []HousingCriterion{{ID: "location"}, {ID: "transport", Weight: checklistPtr(3.0)}, {ID: "layout"}, {ID: "schools"}}
	answers := []HousingAnswer{{CriterionID: "location", State: "complete", Rating: checklistPtr(int32(5))}, {CriterionID: "transport", State: "pending", Rating: checklistPtr(int32(1))}, {CriterionID: "schools", State: "not_applicable"}, {CriterionID: "deleted", State: "complete", Rating: checklistPtr(int32(5))}}
	s := SummarizeHousingEvaluation(criteria, answers)
	if s.Score == nil || *s.Score != 2 || s.Total != 4 || s.Completed != 1 || s.NotApplicable != 1 || s.Rated != 2 || !s.Weighted {
		t.Fatalf("unexpected summary: %+v", s)
	}
	criteria[1].Weight = nil
	s = SummarizeHousingEvaluation(criteria, answers)
	if *s.Score != 3 || s.Weighted {
		t.Fatalf("unexpected unweighted summary: %+v", s)
	}
	for _, items := range [][]HousingCriterion{nil, criteria} {
		s = SummarizeHousingEvaluation(items, nil)
		if s.Score != nil || s.Rated != 0 || s.Completed != 0 {
			t.Fatalf("missing ratings produced score: %+v", s)
		}
	}
	s = SummarizeHousingEvaluation([]HousingCriterion{{ID: "schools"}}, answers)
	if s.Score != nil || s.NotApplicable != 1 {
		t.Fatalf("all N/A: %+v", s)
	}
}
func TestHousingChecklistValidation(t *testing.T) {
	for _, weight := range []float64{0, -1, 1001, math.NaN(), math.Inf(1), math.Inf(-1)} {
		if ValidateHousingCriterion(HousingCriterion{FamilyID: "family", Name: "Location", Weight: &weight}) == nil {
			t.Errorf("accepted weight %v", weight)
		}
	}
	for _, c := range []HousingCriterion{{Name: "Location"}, {FamilyID: "family", Name: " "}, {FamilyID: "family", Name: "Location", DisplayOrder: -1}} {
		if ValidateHousingCriterion(c) == nil {
			t.Errorf("accepted %+v", c)
		}
	}
	for _, a := range []HousingAnswer{{State: "pending"}, {HousingID: "h", CriterionID: "c", State: "invalid"}, {HousingID: "h", CriterionID: "c", State: "complete"}, {HousingID: "h", CriterionID: "c", State: "not_applicable", Rating: checklistPtr(int32(3))}, {HousingID: "h", CriterionID: "c", State: "pending", Rating: checklistPtr(int32(0))}, {HousingID: "h", CriterionID: "c", State: "pending", Rating: checklistPtr(int32(6))}} {
		if ValidateHousingAnswer(a) == nil {
			t.Errorf("accepted %+v", a)
		}
	}
	for _, state := range []string{"pending", "complete", "not_applicable"} {
		a := HousingAnswer{HousingID: "h", CriterionID: "c", State: state}
		if state == "complete" {
			a.Rating = checklistPtr(int32(1))
		}
		if err := ValidateHousingAnswer(a); err != nil {
			t.Errorf("valid answer rejected: %v", err)
		}
	}
}

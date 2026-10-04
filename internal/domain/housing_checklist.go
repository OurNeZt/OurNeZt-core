package domain

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

type HousingCriterion struct {
	ID           ID
	FamilyID     ID
	Name         string
	Description  string
	DisplayOrder int32
	Weight       *float64
}

type HousingAnswer struct {
	HousingID   ID
	CriterionID ID
	State       string
	Rating      *int32
	Notes       string
}

type HousingEvaluationSummary struct {
	Score         *float64
	Total         int32
	Completed     int32
	NotApplicable int32
	Rated         int32
	Weighted      bool
}

func ValidateHousingCriterion(c HousingCriterion) error {
	if strings.TrimSpace(string(c.FamilyID)) == "" || strings.TrimSpace(c.Name) == "" || utf8.RuneCountInString(c.Name) > 120 || utf8.RuneCountInString(c.Description) > 2000 || c.DisplayOrder < 0 {
		return apperror.ErrInvalidArgument
	}
	if c.Weight != nil && (math.IsNaN(*c.Weight) || math.IsInf(*c.Weight, 0) || *c.Weight <= 0 || *c.Weight > 1000) {
		return apperror.ErrInvalidArgument
	}
	return nil
}

func ValidateHousingAnswer(a HousingAnswer) error {
	if a.HousingID == "" || a.CriterionID == "" || utf8.RuneCountInString(a.Notes) > 5000 {
		return apperror.ErrInvalidArgument
	}
	if a.State != "pending" && a.State != "complete" && a.State != "not_applicable" {
		return apperror.ErrInvalidArgument
	}
	if a.Rating != nil && (*a.Rating < 1 || *a.Rating > 5) {
		return apperror.ErrInvalidArgument
	}
	if a.State == "complete" && a.Rating == nil || a.State == "not_applicable" && a.Rating != nil {
		return apperror.ErrInvalidArgument
	}
	return nil
}

// SummarizeHousingEvaluation uses only supplied applicable ratings. Missing weights
// are 1; missing ratings and not-applicable criteria never enter the denominator.
func SummarizeHousingEvaluation(criteria []HousingCriterion, answers []HousingAnswer) HousingEvaluationSummary {
	s := HousingEvaluationSummary{Total: int32(len(criteria))}
	byID := make(map[ID]HousingAnswer, len(answers))
	for _, a := range answers {
		byID[a.CriterionID] = a
	}
	var sum, weights float64
	for _, c := range criteria {
		weight := 1.0
		if c.Weight != nil {
			weight = *c.Weight
			s.Weighted = true
		}
		a := byID[c.ID]
		if a.State == "not_applicable" {
			s.NotApplicable++
			continue
		}
		if a.State == "complete" {
			s.Completed++
		}
		if a.Rating != nil {
			s.Rated++
			sum += float64(*a.Rating) * weight
			weights += weight
		}
	}
	if weights > 0 {
		score := sum / weights
		s.Score = &score
	}
	return s
}

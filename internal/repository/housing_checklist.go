package repository

import (
	"context"
	"github.com/OurNeZt/ournezt-core/internal/domain"
)

type HousingChecklist interface {
	ListHousingCriteria(context.Context, domain.ID, domain.ID) ([]domain.HousingCriterion, error)
	SaveHousingCriterion(context.Context, domain.HousingCriterion, domain.ID) (domain.HousingCriterion, error)
	DeleteHousingCriterion(context.Context, domain.ID, domain.ID, domain.ID) error
	ReorderHousingCriteria(context.Context, domain.ID, []domain.ID, domain.ID) error
	ListHousingAnswers(context.Context, domain.ID, domain.ID) ([]domain.HousingAnswer, error)
	SaveHousingAnswer(context.Context, domain.HousingAnswer, domain.ID) (domain.HousingAnswer, error)
}

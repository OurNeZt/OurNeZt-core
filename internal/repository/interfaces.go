package repository

import (
	"context"

	"github.com/OurNeZt/ournezt-core/internal/domain"
)

type Families interface {
	CreateFamily(ctx context.Context, family domain.Family, ownerID domain.ID) (domain.Family, error)
	GetFamily(ctx context.Context, familyID domain.ID, viewerID domain.ID) (domain.Family, error)
	ListUserFamilies(ctx context.Context, userID domain.ID) ([]domain.Family, error)
}

type People interface {
	CreatePersonProfile(ctx context.Context, profile domain.PersonProfile) (domain.PersonProfile, error)
	GetPersonProfile(ctx context.Context, personID domain.ID, viewerID domain.ID) (domain.PersonProfile, error)
	ListPersonProfilesByFamily(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.PersonProfile, error)
	UpdatePersonProfile(ctx context.Context, profile domain.PersonProfile) (domain.PersonProfile, error)
	DeletePersonProfile(ctx context.Context, personID domain.ID, actorID domain.ID) error
}

type Housing interface {
	CreateHousingOption(ctx context.Context, option domain.HousingOption) (domain.HousingOption, error)
	GetHousingOption(ctx context.Context, housingID domain.ID, viewerID domain.ID) (domain.HousingOption, error)
	ListHousingOptions(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.HousingOption, error)
	UpdateHousingOption(ctx context.Context, option domain.HousingOption) (domain.HousingOption, error)
	DeleteHousingOption(ctx context.Context, housingID domain.ID, actorID domain.ID) error
}


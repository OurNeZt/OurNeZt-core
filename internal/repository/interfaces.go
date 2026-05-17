package repository

import (
	"context"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/domain"
)

type FamilyMember struct {
	FamilyID    domain.ID
	UserID      domain.ID
	DisplayName string
	Role        domain.FamilyRole
}

type Families interface {
	CreateFamily(ctx context.Context, family domain.Family, ownerID domain.ID) (domain.Family, error)
	GetFamily(ctx context.Context, familyID domain.ID, viewerID domain.ID) (domain.Family, error)
	ListUserFamilies(ctx context.Context, userID domain.ID) ([]domain.Family, error)
	JoinFamilyByCode(ctx context.Context, userID domain.ID, codeHash string) (domain.Family, error)
	GenerateFamilyJoinCode(ctx context.Context, actorID domain.ID, familyID domain.ID, codeHash string, expiresAt *time.Time) error
	ListFamilyMembers(ctx context.Context, actorID domain.ID, familyID domain.ID) ([]FamilyMember, error)
}

type People interface {
	CreatePersonProfile(ctx context.Context, profile domain.PersonProfile, actorID domain.ID) (domain.PersonProfile, error)
	GetPersonProfile(ctx context.Context, personID domain.ID, viewerID domain.ID) (domain.PersonProfile, error)
	ListPersonProfilesByFamily(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.PersonProfile, error)
	UpdatePersonProfile(ctx context.Context, profile domain.PersonProfile, actorID domain.ID) (domain.PersonProfile, error)
	DeletePersonProfile(ctx context.Context, personID domain.ID, actorID domain.ID) error
}

type Housing interface {
	CreateHousingOption(ctx context.Context, option domain.HousingOption, actorID domain.ID) (domain.HousingOption, error)
	GetHousingOption(ctx context.Context, housingID domain.ID, viewerID domain.ID) (domain.HousingOption, error)
	ListHousingOptions(ctx context.Context, familyID domain.ID, viewerID domain.ID) ([]domain.HousingOption, error)
	UpdateHousingOption(ctx context.Context, option domain.HousingOption, actorID domain.ID) (domain.HousingOption, error)
	DeleteHousingOption(ctx context.Context, housingID domain.ID, actorID domain.ID) error
}

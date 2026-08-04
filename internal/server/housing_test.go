package server

import (
	"context"
	"testing"

	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/repository"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeHousingRepository struct {
	createInput           domain.HousingOption
	created               domain.HousingOption
	got                   domain.HousingOption
	list                  []domain.HousingOption
	updated               domain.HousingOption
	updateInput           domain.HousingOption
	deletedID             domain.ID
	groups                []domain.HousingGroup
	groupInput            domain.HousingGroup
	createdGroup          domain.HousingGroup
	updatedGroup          domain.HousingGroup
	deletedGroupID        domain.ID
	assignedHousingID     domain.ID
	assignedGroupID       domain.ID
	assignedOption        domain.HousingOption
	visibilityHousingID   domain.ID
	visibilityValue       bool
	visibilityOption      domain.HousingOption
	bulkVisibilityGroupID domain.ID
	bulkVisibilityValue   bool
	bulkVisibilityOptions []domain.HousingOption
}

func (r *fakeHousingRepository) CreateHousingOption(_ context.Context, option domain.HousingOption, _ domain.ID) (domain.HousingOption, error) {
	r.createInput = option
	return r.created, nil
}

func (r *fakeHousingRepository) GetHousingOption(_ context.Context, _ domain.ID, _ domain.ID) (domain.HousingOption, error) {
	return r.got, nil
}

func (r *fakeHousingRepository) ListHousingOptions(_ context.Context, _ domain.ID, _ domain.ID) ([]domain.HousingOption, error) {
	return r.list, nil
}

func (r *fakeHousingRepository) UpdateHousingOption(_ context.Context, option domain.HousingOption, _ domain.ID) (domain.HousingOption, error) {
	r.updateInput = option
	return r.updated, nil
}

func (r *fakeHousingRepository) DeleteHousingOption(_ context.Context, housingID domain.ID, _ domain.ID) error {
	r.deletedID = housingID
	return nil
}

func (r *fakeHousingRepository) CreateHousingGroup(_ context.Context, group domain.HousingGroup, _ domain.ID) (domain.HousingGroup, error) {
	r.groupInput = group
	return r.createdGroup, nil
}

func (r *fakeHousingRepository) ListHousingGroups(_ context.Context, _ domain.ID, _ domain.ID) ([]domain.HousingGroup, error) {
	return r.groups, nil
}

func (r *fakeHousingRepository) UpdateHousingGroup(_ context.Context, group domain.HousingGroup, _ domain.ID) (domain.HousingGroup, error) {
	r.groupInput = group
	return r.updatedGroup, nil
}

func (r *fakeHousingRepository) DeleteHousingGroup(_ context.Context, groupID domain.ID, _ domain.ID) error {
	r.deletedGroupID = groupID
	return nil
}

func (r *fakeHousingRepository) AssignHousingOptionGroup(_ context.Context, housingID domain.ID, groupID domain.ID, _ domain.ID) (domain.HousingOption, error) {
	r.assignedHousingID = housingID
	r.assignedGroupID = groupID
	return r.assignedOption, nil
}

func (r *fakeHousingRepository) UpdateHousingOptionVisibility(_ context.Context, housingID domain.ID, visible bool, _ domain.ID) (domain.HousingOption, error) {
	r.visibilityHousingID = housingID
	r.visibilityValue = visible
	return r.visibilityOption, nil
}

func (r *fakeHousingRepository) BulkUpdateHousingGroupVisibility(_ context.Context, groupID domain.ID, visible bool, _ domain.ID) ([]domain.HousingOption, error) {
	r.bulkVisibilityGroupID = groupID
	r.bulkVisibilityValue = visible
	return r.bulkVisibilityOptions, nil
}

func TestHousingServerCreateOption(t *testing.T) {
	repo := &fakeHousingRepository{
		created: domain.HousingOption{ID: "housing_1", FamilyID: "family_1", Name: "BTO"},
	}
	peopleRepo := &fakePeopleRepository{
		list: []domain.PersonProfile{
			{ID: "person_1", FamilyID: "family_1", GrossMonthlyIncomeCents: 320000},
			{ID: "person_2", FamilyID: "family_1", GrossMonthlyIncomeCents: 180000},
		},
	}
	server := NewHousingServer(repo, peopleRepo)

	response, err := server.CreateHousingOption(context.Background(), &ourneztv1.HousingOption{
		FamilyId:              "family_1",
		Name:                  "BTO",
		HousingType:           "bto",
		LoanType:              "hdb",
		PurchasePriceCents:    45000000,
		InterestRateBps:       260,
		LoanTenureMonths:      300,
		DownpaymentPercentBps: 2000,
	})
	if err != nil {
		t.Fatalf("CreateHousingOption returned error: %v", err)
	}
	if response.GetId() != "housing_1" {
		t.Fatalf("id = %q, want housing_1", response.GetId())
	}
	if repo.createInput.Name != "BTO" {
		t.Fatalf("create input name = %q, want BTO", repo.createInput.Name)
	}
	if repo.createInput.GrantAmountCents != 6500000 {
		t.Fatalf("create input grant amount = %d, want 6500000", repo.createInput.GrantAmountCents)
	}
	if !repo.createInput.VisibleOnDashboard {
		t.Fatal("create input visible on dashboard = false, want true")
	}
}

func TestHousingServerCalculateAffordability(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{}, &fakePeopleRepository{})

	response, err := server.CalculateHousingAffordability(context.Background(), &ourneztv1.CalculateHousingAffordabilityRequest{
		HousingOption: &ourneztv1.HousingOption{
			Id:                    "housing_1",
			FamilyId:              "family_1",
			Name:                  "BTO",
			HousingType:           "bto",
			LoanType:              "hdb",
			PurchasePriceCents:    45000000,
			InterestRateBps:       260,
			LoanTenureMonths:      300,
			DownpaymentPercentBps: 2000,
		},
		CashSavingsCents:     5000000,
		CpfOaCents:           3000000,
		TakeHomeCents:        650000,
		MonthlyExpensesCents: 200000,
	})
	if err != nil {
		t.Fatalf("CalculateHousingAffordability returned error: %v", err)
	}
	if response.GetHousingOptionId() != "housing_1" {
		t.Fatalf("housing option id = %q, want housing_1", response.GetHousingOptionId())
	}
	if response.GetMonthlyMortgageCents() <= 0 {
		t.Fatalf("monthly mortgage = %d, want > 0", response.GetMonthlyMortgageCents())
	}
	if response.GetInitialDownpaymentCents() != 2250000 {
		t.Fatalf("initial downpayment = %d, want 2250000", response.GetInitialDownpaymentCents())
	}
	if response.GetFinalDownpaymentCents() != 6750000 {
		t.Fatalf("final downpayment = %d, want 6750000", response.GetFinalDownpaymentCents())
	}
}

func TestHousingServerRejectsLoanTenureAboveHousingTypeCap(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{}, &fakePeopleRepository{})

	tests := []struct {
		name          string
		housingType   string
		tenureMonths  int32
		wantErrorCode codes.Code
	}{
		{
			name:          "HDB above 30 years",
			housingType:   "bto",
			tenureMonths:  31 * 12,
			wantErrorCode: codes.InvalidArgument,
		},
		{
			name:          "non-HDB above 35 years",
			housingType:   "private_condo",
			tenureMonths:  36 * 12,
			wantErrorCode: codes.InvalidArgument,
		},
		{
			name:         "non-HDB at 35 years",
			housingType:  "private_condo",
			tenureMonths: 35 * 12,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.CalculateHousingAffordability(context.Background(), &ourneztv1.CalculateHousingAffordabilityRequest{
				HousingOption: &ourneztv1.HousingOption{
					Id:                    "housing_1",
					FamilyId:              "family_1",
					Name:                  "Option",
					HousingType:           tc.housingType,
					LoanType:              "bank",
					PurchasePriceCents:    45000000,
					InterestRateBps:       260,
					LoanTenureMonths:      tc.tenureMonths,
					DownpaymentPercentBps: 2500,
				},
				TakeHomeCents: 650000,
			})
			if tc.wantErrorCode == codes.OK {
				if err != nil {
					t.Fatalf("CalculateHousingAffordability returned error: %v", err)
				}
				return
			}
			if status.Code(err) != tc.wantErrorCode {
				t.Fatalf("status code = %v, want %v", status.Code(err), tc.wantErrorCode)
			}
		})
	}
}

func TestHousingServerRejectsNonHDBPropertyHDBOnlyValues(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{}, &fakePeopleRepository{})

	tests := []struct {
		name        string
		housingType string
		mutate      func(*ourneztv1.HousingOption)
	}{
		{
			name:        "landed HDB loan",
			housingType: "landed",
			mutate: func(option *ourneztv1.HousingOption) {
				option.LoanType = "hdb"
			},
		},
		{
			name:        "other grant amount",
			housingType: "other",
			mutate: func(option *ourneztv1.HousingOption) {
				option.GrantAmountCents = 1000000
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			option := &ourneztv1.HousingOption{
				Id:                    "housing_1",
				FamilyId:              "family_1",
				Name:                  "Non-HDB Option",
				HousingType:           tc.housingType,
				LoanType:              "bank",
				PurchasePriceCents:    120000000,
				InterestRateBps:       360,
				LoanTenureMonths:      35 * 12,
				DownpaymentPercentBps: 2500,
			}
			tc.mutate(option)

			_, err := server.CalculateHousingAffordability(context.Background(), &ourneztv1.CalculateHousingAffordabilityRequest{
				HousingOption: option,
				TakeHomeCents: 1000000,
			})
			if status.Code(err) != codes.InvalidArgument {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
			}
		})
	}
}

func TestHousingServerDeleteOption(t *testing.T) {
	repo := &fakeHousingRepository{}
	server := NewHousingServer(repo, &fakePeopleRepository{})

	_, err := server.DeleteHousingOption(context.Background(), &ourneztv1.DeleteHousingOptionRequest{
		ActorUserId: "user_1",
		HousingId:   "housing_9",
	})
	if err != nil {
		t.Fatalf("DeleteHousingOption returned error: %v", err)
	}
	if repo.deletedID != "housing_9" {
		t.Fatalf("deleted id = %q, want housing_9", repo.deletedID)
	}
}

func TestHousingServerUpdateOptionPreservesGroupAndVisibilityWhenFieldsOmitted(t *testing.T) {
	repo := &fakeHousingRepository{
		got: domain.HousingOption{
			ID:                 "housing_1",
			FamilyID:           "family_1",
			GroupID:            "group_1",
			VisibleOnDashboard: false,
		},
		updated: domain.HousingOption{
			ID:                 "housing_1",
			FamilyID:           "family_1",
			GroupID:            "group_1",
			VisibleOnDashboard: false,
		},
	}
	server := NewHousingServer(repo, &fakePeopleRepository{})

	_, err := server.UpdateHousingOption(context.Background(), &ourneztv1.HousingOption{
		Id:                    "housing_1",
		FamilyId:              "family_1",
		Name:                  "Updated Option",
		HousingType:           "bto",
		LoanType:              "hdb",
		PurchasePriceCents:    45000000,
		InterestRateBps:       260,
		LoanTenureMonths:      300,
		DownpaymentPercentBps: 2000,
	})
	if err != nil {
		t.Fatalf("UpdateHousingOption returned error: %v", err)
	}
	if repo.updateInput.GroupID != "group_1" {
		t.Fatalf("update group id = %q, want group_1", repo.updateInput.GroupID)
	}
	if repo.updateInput.VisibleOnDashboard {
		t.Fatal("update visibility = true, want false")
	}
}

func TestHousingServerCreateGroup(t *testing.T) {
	repo := &fakeHousingRepository{
		createdGroup: domain.HousingGroup{ID: "group_1", FamilyID: "family_1", Name: "June BTO 2026"},
	}
	server := NewHousingServer(repo, &fakePeopleRepository{})

	response, err := server.CreateHousingGroup(context.Background(), &ourneztv1.HousingGroup{
		FamilyId: "family_1",
		Name:     "June BTO 2026",
	})
	if err != nil {
		t.Fatalf("CreateHousingGroup returned error: %v", err)
	}
	if response.GetId() != "group_1" {
		t.Fatalf("group id = %q, want group_1", response.GetId())
	}
	if repo.groupInput.Name != "June BTO 2026" {
		t.Fatalf("group input name = %q, want June BTO 2026", repo.groupInput.Name)
	}
}

func TestHousingServerListGroups(t *testing.T) {
	repo := &fakeHousingRepository{
		groups: []domain.HousingGroup{
			{ID: "group_1", FamilyID: "family_1", Name: "Resale"},
			{ID: "group_2", FamilyID: "family_1", Name: "EC"},
		},
	}
	server := NewHousingServer(repo, &fakePeopleRepository{})

	response, err := server.ListHousingGroups(context.Background(), &ourneztv1.ListHousingGroupsRequest{
		ViewerUserId: "user_1",
		FamilyId:     "family_1",
	})
	if err != nil {
		t.Fatalf("ListHousingGroups returned error: %v", err)
	}
	if len(response.GetHousingGroups()) != 2 {
		t.Fatalf("housing groups len = %d, want 2", len(response.GetHousingGroups()))
	}
}

func TestHousingServerAssignHousingOptionGroup(t *testing.T) {
	repo := &fakeHousingRepository{
		assignedOption: domain.HousingOption{
			ID:                 "housing_1",
			FamilyID:           "family_1",
			GroupID:            "group_1",
			VisibleOnDashboard: true,
		},
	}
	server := NewHousingServer(repo, &fakePeopleRepository{})

	response, err := server.AssignHousingOptionGroup(context.Background(), &ourneztv1.AssignHousingOptionGroupRequest{
		ActorUserId:    "user_1",
		HousingId:      "housing_1",
		HousingGroupId: "group_1",
	})
	if err != nil {
		t.Fatalf("AssignHousingOptionGroup returned error: %v", err)
	}
	if repo.assignedHousingID != "housing_1" {
		t.Fatalf("assigned housing id = %q, want housing_1", repo.assignedHousingID)
	}
	if repo.assignedGroupID != "group_1" {
		t.Fatalf("assigned group id = %q, want group_1", repo.assignedGroupID)
	}
	if response.GetHousingGroupId() != "group_1" {
		t.Fatalf("response housing group id = %q, want group_1", response.GetHousingGroupId())
	}
}

func TestHousingServerUpdateHousingOptionVisibility(t *testing.T) {
	repo := &fakeHousingRepository{
		visibilityOption: domain.HousingOption{
			ID:                 "housing_1",
			FamilyID:           "family_1",
			VisibleOnDashboard: false,
		},
	}
	server := NewHousingServer(repo, &fakePeopleRepository{})

	response, err := server.UpdateHousingOptionVisibility(context.Background(), &ourneztv1.UpdateHousingOptionVisibilityRequest{
		ActorUserId:        "user_1",
		HousingId:          "housing_1",
		VisibleOnDashboard: false,
	})
	if err != nil {
		t.Fatalf("UpdateHousingOptionVisibility returned error: %v", err)
	}
	if repo.visibilityHousingID != "housing_1" {
		t.Fatalf("visibility housing id = %q, want housing_1", repo.visibilityHousingID)
	}
	if repo.visibilityValue {
		t.Fatal("visibility value = true, want false")
	}
	if response.GetVisibleOnDashboard() {
		t.Fatal("response visible on dashboard = true, want false")
	}
}

func TestHousingServerBulkUpdateHousingGroupVisibility(t *testing.T) {
	repo := &fakeHousingRepository{
		bulkVisibilityOptions: []domain.HousingOption{
			{ID: "housing_1", FamilyID: "family_1", GroupID: "group_1", VisibleOnDashboard: false},
			{ID: "housing_2", FamilyID: "family_1", GroupID: "group_1", VisibleOnDashboard: false},
		},
	}
	server := NewHousingServer(repo, &fakePeopleRepository{})

	response, err := server.BulkUpdateHousingGroupVisibility(context.Background(), &ourneztv1.BulkUpdateHousingGroupVisibilityRequest{
		ActorUserId:        "user_1",
		HousingGroupId:     "group_1",
		VisibleOnDashboard: false,
	})
	if err != nil {
		t.Fatalf("BulkUpdateHousingGroupVisibility returned error: %v", err)
	}
	if repo.bulkVisibilityGroupID != "group_1" {
		t.Fatalf("bulk visibility group id = %q, want group_1", repo.bulkVisibilityGroupID)
	}
	if repo.bulkVisibilityValue {
		t.Fatal("bulk visibility value = true, want false")
	}
	if len(response.GetHousingOptions()) != 2 {
		t.Fatalf("housing options len = %d, want 2", len(response.GetHousingOptions()))
	}
}

func TestHousingServerEstimateHousingGrant(t *testing.T) {
	server := NewHousingServer(&fakeHousingRepository{}, &fakePeopleRepository{
		list: []domain.PersonProfile{
			{ID: "person_1", FamilyID: "family_1", GrossMonthlyIncomeCents: 300000},
			{ID: "person_2", FamilyID: "family_1", GrossMonthlyIncomeCents: 200000},
		},
	})

	response, err := server.EstimateHousingGrant(context.Background(), &ourneztv1.EstimateHousingGrantRequest{
		ViewerUserId: "user_1",
		FamilyId:     "family_1",
		HousingType:  "bto",
	})
	if err != nil {
		t.Fatalf("EstimateHousingGrant returned error: %v", err)
	}
	if response.GetHouseholdGrossMonthlyIncomeCents() != 500000 {
		t.Fatalf("household gross = %d, want 500000", response.GetHouseholdGrossMonthlyIncomeCents())
	}
	if response.GetGrantAmountCents() != 6500000 {
		t.Fatalf("grant amount = %d, want 6500000", response.GetGrantAmountCents())
	}
	if !response.GetEligible() {
		t.Fatalf("eligible = false, want true")
	}
}

var _ repository.Housing = (*fakeHousingRepository)(nil)

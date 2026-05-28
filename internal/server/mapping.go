package server

import (
	"strings"
	"time"

	"github.com/OurNeZt/ournezt-core/internal/calculation"
	"github.com/OurNeZt/ournezt-core/internal/domain"
	ourneztv1 "github.com/OurNeZt/ournezt-core/internal/gen/proto/ournezt/v1"
	"github.com/OurNeZt/ournezt-core/internal/platform/apperror"
)

func requireID(value string) (domain.ID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperror.ErrInvalidArgument
	}
	return domain.ID(trimmed), nil
}

func parseOptionalDate(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, apperror.ErrInvalidArgument
	}
	return &parsed, nil
}

func formatDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}

func userToProto(user domain.User) *ourneztv1.User {
	return &ourneztv1.User{
		Id:                 string(user.ID),
		Email:              user.Email,
		DisplayName:        user.DisplayName,
		Role:               string(user.Role),
		Disabled:           user.DisabledAt != nil,
		MustChangePassword: user.MustChangePassword,
	}
}

func familyToProto(family domain.Family) *ourneztv1.Family {
	return &ourneztv1.Family{
		Id:         string(family.ID),
		Name:       family.Name,
		FamilyType: string(family.Type),
	}
}

func personFromProto(in *ourneztv1.PersonProfile) (domain.PersonProfile, error) {
	if in == nil {
		return domain.PersonProfile{}, apperror.ErrInvalidArgument
	}
	expectedStart, err := parseOptionalDate(in.GetExpectedIncomeStartDate())
	if err != nil {
		return domain.PersonProfile{}, err
	}
	graduationDate, err := parseOptionalDate(in.GetGraduationDate())
	if err != nil {
		return domain.PersonProfile{}, err
	}
	ordDate, err := parseOptionalDate(in.GetOrdDate())
	if err != nil {
		return domain.PersonProfile{}, err
	}

	return domain.PersonProfile{
		ID:                        domain.ID(strings.TrimSpace(in.GetId())),
		FamilyID:                  domain.ID(strings.TrimSpace(in.GetFamilyId())),
		Name:                      strings.TrimSpace(in.GetName()),
		Age:                       int(in.GetAge()),
		RelationshipLabel:         strings.TrimSpace(in.GetRelationshipLabel()),
		EmploymentStatus:          domain.EmploymentStatus(strings.TrimSpace(in.GetEmploymentStatus())),
		GrossMonthlyIncomeCents:   in.GetGrossMonthlyIncomeCents(),
		ExpectedFutureIncomeCents: in.GetExpectedFutureIncomeCents(),
		ExpectedIncomeStartDate:   expectedStart,
		GraduationDate:            graduationDate,
		ORDDate:                   ordDate,
		CashSavingsCents:          in.GetCashSavingsCents(),
		CPFOACents:                in.GetCpfOaCents(),
		CPFSACents:                in.GetCpfSaCents(),
		CPFMACents:                in.GetCpfMaCents(),
		MonthlyExpensesCents:      in.GetMonthlyExpensesCents(),
	}, nil
}

func personToProto(person domain.PersonProfile) *ourneztv1.PersonProfile {
	return &ourneztv1.PersonProfile{
		Id:                        string(person.ID),
		FamilyId:                  string(person.FamilyID),
		Name:                      person.Name,
		Age:                       int32(person.Age),
		RelationshipLabel:         person.RelationshipLabel,
		EmploymentStatus:          string(person.EmploymentStatus),
		GrossMonthlyIncomeCents:   person.GrossMonthlyIncomeCents,
		ExpectedFutureIncomeCents: person.ExpectedFutureIncomeCents,
		ExpectedIncomeStartDate:   formatDate(person.ExpectedIncomeStartDate),
		GraduationDate:            formatDate(person.GraduationDate),
		OrdDate:                   formatDate(person.ORDDate),
		CashSavingsCents:          person.CashSavingsCents,
		CpfOaCents:                person.CPFOACents,
		CpfSaCents:                person.CPFSACents,
		CpfMaCents:                person.CPFMACents,
		MonthlyExpensesCents:      person.MonthlyExpensesCents,
	}
}

func personIncomeHistoryEntryToProto(entry domain.PersonIncomeHistoryEntry) *ourneztv1.IncomeHistoryEntry {
	return &ourneztv1.IncomeHistoryEntry{
		PersonId:                  string(entry.PersonID),
		PersonName:                entry.PersonName,
		GrossMonthlyIncomeCents:   entry.GrossMonthlyIncomeCents,
		ExpectedFutureIncomeCents: entry.ExpectedFutureIncomeCents,
		RecordedAt:                entry.RecordedAt.UTC().Format(time.RFC3339),
	}
}

func housingFromProto(in *ourneztv1.HousingOption) (domain.HousingOption, error) {
	if in == nil {
		return domain.HousingOption{}, apperror.ErrInvalidArgument
	}

	keyDate, err := parseOptionalDate(in.GetExpectedKeyCollectionDate())
	if err != nil {
		return domain.HousingOption{}, err
	}

	return domain.HousingOption{
		ID:                        domain.ID(strings.TrimSpace(in.GetId())),
		FamilyID:                  domain.ID(strings.TrimSpace(in.GetFamilyId())),
		Name:                      strings.TrimSpace(in.GetName()),
		Type:                      domain.HousingType(strings.TrimSpace(in.GetHousingType())),
		Location:                  strings.TrimSpace(in.GetLocation()),
		UnitType:                  strings.TrimSpace(in.GetUnitType()),
		PurchasePriceCents:        in.GetPurchasePriceCents(),
		GrantAmountCents:          in.GetGrantAmountCents(),
		LoanType:                  domain.LoanType(strings.TrimSpace(in.GetLoanType())),
		LoanAmountCents:           in.GetLoanAmountCents(),
		InterestRateBps:           in.GetInterestRateBps(),
		LoanTenureMonths:          int(in.GetLoanTenureMonths()),
		DownpaymentPercentBps:     in.GetDownpaymentPercentBps(),
		RenovationBudgetCents:     in.GetRenovationBudgetCents(),
		FurnitureBudgetCents:      in.GetFurnitureBudgetCents(),
		LegalFeesCents:            in.GetLegalFeesCents(),
		BuyerStampDutyCents:       in.GetBuyerStampDutyCents(),
		MonthlyMaintenanceCents:   in.GetMonthlyMaintenanceCents(),
		ExpectedKeyCollectionDate: keyDate,
	}, nil
}

func housingToProto(option domain.HousingOption) *ourneztv1.HousingOption {
	return &ourneztv1.HousingOption{
		Id:                        string(option.ID),
		FamilyId:                  string(option.FamilyID),
		Name:                      option.Name,
		HousingType:               string(option.Type),
		Location:                  option.Location,
		UnitType:                  option.UnitType,
		PurchasePriceCents:        option.PurchasePriceCents,
		GrantAmountCents:          option.GrantAmountCents,
		LoanType:                  string(option.LoanType),
		LoanAmountCents:           option.LoanAmountCents,
		InterestRateBps:           option.InterestRateBps,
		LoanTenureMonths:          int32(option.LoanTenureMonths),
		DownpaymentPercentBps:     option.DownpaymentPercentBps,
		RenovationBudgetCents:     option.RenovationBudgetCents,
		FurnitureBudgetCents:      option.FurnitureBudgetCents,
		LegalFeesCents:            option.LegalFeesCents,
		BuyerStampDutyCents:       option.BuyerStampDutyCents,
		MonthlyMaintenanceCents:   option.MonthlyMaintenanceCents,
		ExpectedKeyCollectionDate: formatDate(option.ExpectedKeyCollectionDate),
	}
}

func incomeBreakdownToProto(in calculation.IncomeBreakdown) *ourneztv1.IncomeBreakdown {
	return &ourneztv1.IncomeBreakdown{
		PersonId:                  string(in.PersonID),
		CurrentGrossIncomeCents:   in.CurrentGrossIncomeCents,
		ProjectedGrossIncomeCents: in.ProjectedGrossIncomeCents,
		EmployeeCpfCents:          in.EmployeeCPFCents,
		EmployerCpfCents:          in.EmployerCPFCents,
		TakeHomeIncomeCents:       in.TakeHomeIncomeCents,
		MonthlyExpensesCents:      in.MonthlyExpensesCents,
		MonthlySurplusCents:       in.MonthlySurplusCents,
		CpfPlanningNote:           in.CPFPlanningNote,
	}
}

func householdIncomeToProto(in calculation.HouseholdIncomeSummary) *ourneztv1.HouseholdIncomeSummary {
	return &ourneztv1.HouseholdIncomeSummary{
		CurrentGrossIncomeCents:   in.CurrentGrossIncomeCents,
		ProjectedGrossIncomeCents: in.ProjectedGrossIncomeCents,
		TakeHomeIncomeCents:       in.TakeHomeIncomeCents,
		EmployeeCpfCents:          in.EmployeeCPFCents,
		EmployerCpfCents:          in.EmployerCPFCents,
		MonthlyExpensesCents:      in.MonthlyExpensesCents,
		MonthlySurplusCents:       in.MonthlySurplusCents,
		CurrentCpfOaCents:         in.CurrentCPFOACents,
		CurrentCpfSaCents:         in.CurrentCPFSACents,
		CurrentCpfMaCents:         in.CurrentCPFMACents,
		MayNeedDeferredAssessment: in.MayNeedDeferredAssessment,
	}
}

func cpfContributionToProto(in calculation.CPFContribution) *ourneztv1.CPFContribution {
	return &ourneztv1.CPFContribution{
		CpfApplicableWageCents: in.CPFApplicableWageCents,
		EmployeeCents:          in.EmployeeCents,
		EmployerCents:          in.EmployerCents,
		TotalCents:             in.TotalCents,
		OrdinaryAccountCents:   in.OrdinaryAccountCents,
		SpecialAccountCents:    in.SpecialAccountCents,
		MedisaveAccountCents:   in.MedisaveAccountCents,
	}
}

func housingAffordabilityToProto(in calculation.HousingAffordability) *ourneztv1.HousingAffordability {
	return &ourneztv1.HousingAffordability{
		HousingOptionId:                 string(in.HousingOptionID),
		NetPurchasePriceCents:           in.NetPurchasePriceCents,
		RequiredDownpaymentCents:        in.RequiredDownpaymentCents,
		UpfrontCostCents:                in.UpfrontCostCents,
		EstimatedLoanAmountCents:        in.EstimatedLoanAmountCents,
		MonthlyMortgageCents:            in.MonthlyMortgageCents,
		CpfUsedCents:                    in.CPFUsedCents,
		CashTopUpRequiredCents:          in.CashTopUpRequiredCents,
		RemainingCashCents:              in.RemainingCashCents,
		RemainingCpfOaCents:             in.RemainingCPFOACents,
		MonthlyHousingCostCents:         in.MonthlyHousingCostCents,
		MonthlySurplusAfterHousingCents: in.MonthlySurplusAfterHousingCents,
		Rating:                          string(in.Rating),
	}
}

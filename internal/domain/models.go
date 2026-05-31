package domain

import "time"

type ID string

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

type User struct {
	ID                 ID
	Email              string
	DisplayName        string
	Role               UserRole
	PasswordHash       string
	MustChangePassword bool
	DisabledAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type FamilyType string

const (
	FamilyTypeSingle          FamilyType = "single"
	FamilyTypeCouple          FamilyType = "couple"
	FamilyTypeFamily          FamilyType = "family"
	FamilyTypeSharedHousehold FamilyType = "shared_household"
)

type FamilyRole string

const (
	FamilyRoleOwner  FamilyRole = "owner"
	FamilyRoleAdmin  FamilyRole = "admin"
	FamilyRoleMember FamilyRole = "member"
	FamilyRoleViewer FamilyRole = "viewer"
)

type Family struct {
	ID        ID
	Name      string
	Type      FamilyType
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FamilyMember struct {
	FamilyID ID
	UserID   ID
	Role     FamilyRole
	JoinedAt time.Time
}

type EmploymentStatus string

const (
	EmploymentFullTime       EmploymentStatus = "full_time_employee"
	EmploymentPartTime       EmploymentStatus = "part_time_employee"
	EmploymentSelfEmployed   EmploymentStatus = "self_employed"
	EmploymentStudent        EmploymentStatus = "student"
	EmploymentFullTimeNSF    EmploymentStatus = "full_time_nsf"
	EmploymentUnemployed     EmploymentStatus = "unemployed"
	EmploymentFutureEmployee EmploymentStatus = "future_employee"
	EmploymentOther          EmploymentStatus = "other"
)

type PersonProfile struct {
	ID                        ID
	FamilyID                  ID
	LinkedUserID              ID
	Name                      string
	Age                       int
	RelationshipLabel         string
	EmploymentStatus          EmploymentStatus
	GrossMonthlyIncomeCents   int64
	ExpectedFutureIncomeCents int64
	ExpectedIncomeStartDate   *time.Time
	GraduationDate            *time.Time
	ORDDate                   *time.Time
	CashSavingsCents          int64
	CPFOACents                int64
	CPFSACents                int64
	CPFMACents                int64
	MonthlyExpensesCents      int64
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type PersonIncomeHistoryEntry struct {
	PersonID                  ID
	PersonName                string
	GrossMonthlyIncomeCents   int64
	ExpectedFutureIncomeCents int64
	RecordedAt                time.Time
}

type HousingType string

const (
	HousingTypeBTO       HousingType = "bto"
	HousingTypeResaleHDB HousingType = "resale_hdb"
	HousingTypeExecutive HousingType = "executive_condo"
	HousingTypePrivate   HousingType = "private_condo"
	HousingTypeLanded    HousingType = "landed"
	HousingTypeOther     HousingType = "other"
)

type LoanType string

const (
	LoanTypeHDB  LoanType = "hdb"
	LoanTypeBank LoanType = "bank"
	LoanTypeCash LoanType = "cash"
)

type HousingOption struct {
	ID                        ID
	FamilyID                  ID
	Name                      string
	Type                      HousingType
	Location                  string
	UnitType                  string
	PurchasePriceCents        int64
	GrantAmountCents          int64
	LoanType                  LoanType
	LoanAmountCents           int64
	InterestRateBps           int64
	LoanTenureMonths          int
	DownpaymentPercentBps     int64
	RenovationBudgetCents     int64
	FurnitureBudgetCents      int64
	LegalFeesCents            int64
	BuyerStampDutyCents       int64
	MonthlyMaintenanceCents   int64
	ExpectedKeyCollectionDate *time.Time
	DIAIncomeOverrides        []HousingDIAIncomeOverride
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type HousingDIAIncomeOverride struct {
	PersonID             ID
	ProjectedIncomeCents int64
}

type AffordabilityRating string

const (
	AffordabilityComfortable    AffordabilityRating = "comfortable"
	AffordabilityManageable     AffordabilityRating = "manageable"
	AffordabilityTight          AffordabilityRating = "tight"
	AffordabilityRisky          AffordabilityRating = "risky"
	AffordabilityNotRecommended AffordabilityRating = "not_recommended"
)

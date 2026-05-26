CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    password_hash TEXT NOT NULL,
    disabled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE families (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    family_type TEXT NOT NULL CHECK (family_type IN ('single', 'couple', 'family', 'shared_household')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE family_members (
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (family_id, user_id)
);

CREATE TABLE family_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE person_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    age INTEGER NOT NULL CHECK (age >= 0 AND age <= 120),
    relationship_label TEXT NOT NULL DEFAULT '',
    employment_status TEXT NOT NULL CHECK (employment_status IN (
        'full_time_employee',
        'part_time_employee',
        'self_employed',
        'student',
        'full_time_nsf',
        'unemployed',
        'future_employee',
        'other'
    )),
    gross_monthly_income_cents BIGINT NOT NULL DEFAULT 0 CHECK (gross_monthly_income_cents >= 0),
    expected_future_income_cents BIGINT NOT NULL DEFAULT 0 CHECK (expected_future_income_cents >= 0),
    expected_income_start_date DATE,
    graduation_date DATE,
    ord_date DATE,
    cash_savings_cents BIGINT NOT NULL DEFAULT 0 CHECK (cash_savings_cents >= 0),
    cpf_oa_cents BIGINT NOT NULL DEFAULT 0 CHECK (cpf_oa_cents >= 0),
    cpf_sa_cents BIGINT NOT NULL DEFAULT 0 CHECK (cpf_sa_cents >= 0),
    cpf_ma_cents BIGINT NOT NULL DEFAULT 0 CHECK (cpf_ma_cents >= 0),
    monthly_expenses_cents BIGINT NOT NULL DEFAULT 0 CHECK (monthly_expenses_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE cpf_profiles (
    person_profile_id UUID PRIMARY KEY REFERENCES person_profiles(id) ON DELETE CASCADE,
    ordinary_account_cents BIGINT NOT NULL DEFAULT 0 CHECK (ordinary_account_cents >= 0),
    special_account_cents BIGINT NOT NULL DEFAULT 0 CHECK (special_account_cents >= 0),
    medisave_account_cents BIGINT NOT NULL DEFAULT 0 CHECK (medisave_account_cents >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE housing_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    housing_type TEXT NOT NULL CHECK (housing_type IN ('bto', 'resale_hdb', 'executive_condo', 'private_condo', 'landed', 'other')),
    location TEXT NOT NULL DEFAULT '',
    unit_type TEXT NOT NULL DEFAULT '',
    purchase_price_cents BIGINT NOT NULL CHECK (purchase_price_cents >= 0),
    grant_amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (grant_amount_cents >= 0),
    loan_type TEXT NOT NULL CHECK (loan_type IN ('hdb', 'bank', 'cash')),
    loan_amount_cents BIGINT NOT NULL DEFAULT 0 CHECK (loan_amount_cents >= 0),
    interest_rate_bps INTEGER NOT NULL DEFAULT 0 CHECK (interest_rate_bps >= 0),
    loan_tenure_months INTEGER NOT NULL DEFAULT 0 CHECK (loan_tenure_months >= 0),
    downpayment_percent_bps INTEGER NOT NULL DEFAULT 0 CHECK (downpayment_percent_bps >= 0),
    renovation_budget_cents BIGINT NOT NULL DEFAULT 0 CHECK (renovation_budget_cents >= 0),
    furniture_budget_cents BIGINT NOT NULL DEFAULT 0 CHECK (furniture_budget_cents >= 0),
    legal_fees_cents BIGINT NOT NULL DEFAULT 0 CHECK (legal_fees_cents >= 0),
    buyer_stamp_duty_cents BIGINT NOT NULL DEFAULT 0 CHECK (buyer_stamp_duty_cents >= 0),
    monthly_maintenance_cents BIGINT NOT NULL DEFAULT 0 CHECK (monthly_maintenance_cents >= 0),
    expected_key_collection_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_user_id_idx ON sessions(user_id);
CREATE INDEX family_members_user_id_idx ON family_members(user_id);
CREATE INDEX person_profiles_family_id_idx ON person_profiles(family_id);
CREATE INDEX housing_options_family_id_idx ON housing_options(family_id);


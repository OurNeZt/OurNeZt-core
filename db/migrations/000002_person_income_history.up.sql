CREATE TABLE person_income_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_profile_id UUID NOT NULL REFERENCES person_profiles(id) ON DELETE CASCADE,
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    person_name TEXT NOT NULL DEFAULT '',
    gross_monthly_income_cents BIGINT NOT NULL DEFAULT 0 CHECK (gross_monthly_income_cents >= 0),
    expected_future_income_cents BIGINT NOT NULL DEFAULT 0 CHECK (expected_future_income_cents >= 0),
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX person_income_history_family_recorded_idx
    ON person_income_history(family_id, recorded_at);

CREATE INDEX person_income_history_person_recorded_idx
    ON person_income_history(person_profile_id, recorded_at);

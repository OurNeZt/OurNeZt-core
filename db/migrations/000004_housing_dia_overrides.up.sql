ALTER TABLE housing_options
ADD COLUMN dia_income_overrides JSONB NOT NULL DEFAULT '[]'::jsonb;

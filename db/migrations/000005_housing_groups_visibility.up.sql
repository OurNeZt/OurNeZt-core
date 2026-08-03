CREATE TABLE housing_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX housing_groups_family_id_name_key ON housing_groups (family_id, lower(name));
CREATE INDEX housing_groups_family_id_idx ON housing_groups(family_id);

ALTER TABLE housing_options
    ADD COLUMN housing_group_id UUID REFERENCES housing_groups(id) ON DELETE SET NULL,
    ADD COLUMN visible_on_dashboard BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX housing_options_housing_group_id_idx ON housing_options(housing_group_id);

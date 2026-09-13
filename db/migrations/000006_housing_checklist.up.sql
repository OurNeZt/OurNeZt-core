-- One ordered checklist per family. Retired criteria retain their historical answers.
CREATE TABLE housing_checklist_criteria (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id UUID NOT NULL REFERENCES families(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(btrim(name)) BETWEEN 1 AND 120),
    description TEXT NOT NULL DEFAULT '' CHECK (length(description) <= 2000),
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    weight DOUBLE PRECISION CHECK (weight > 0 AND weight <= 1000),
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, family_id)
);
CREATE INDEX housing_checklist_active_idx ON housing_checklist_criteria(family_id, display_order) WHERE deleted_at IS NULL;
ALTER TABLE housing_options ADD CONSTRAINT housing_options_id_family_key UNIQUE (id, family_id);
CREATE TABLE housing_checklist_answers (
    housing_id UUID NOT NULL,
    criterion_id UUID NOT NULL,
    family_id UUID NOT NULL,
    state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending', 'complete', 'not_applicable')),
    rating INTEGER CHECK (rating BETWEEN 1 AND 5),
    notes TEXT NOT NULL DEFAULT '' CHECK (length(notes) <= 5000),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (housing_id, criterion_id),
    FOREIGN KEY (housing_id, family_id) REFERENCES housing_options(id, family_id) ON DELETE CASCADE,
    FOREIGN KEY (criterion_id, family_id) REFERENCES housing_checklist_criteria(id, family_id) ON DELETE CASCADE,
    CHECK (state <> 'complete' OR rating IS NOT NULL),
    CHECK (state <> 'not_applicable' OR rating IS NULL)
);
CREATE INDEX housing_checklist_answers_family_idx ON housing_checklist_answers(family_id);

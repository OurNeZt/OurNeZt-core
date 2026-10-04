ALTER TABLE housing_options
    ADD COLUMN notes TEXT NOT NULL DEFAULT '',
    ADD CONSTRAINT housing_options_notes_length CHECK (char_length(notes) <= 10000);

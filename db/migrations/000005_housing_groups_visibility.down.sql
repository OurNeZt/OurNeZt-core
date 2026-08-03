DROP INDEX IF EXISTS housing_options_housing_group_id_idx;

ALTER TABLE housing_options
    DROP COLUMN IF EXISTS visible_on_dashboard,
    DROP COLUMN IF EXISTS housing_group_id;

DROP INDEX IF EXISTS housing_groups_family_id_idx;
DROP INDEX IF EXISTS housing_groups_family_id_name_key;
DROP TABLE IF EXISTS housing_groups;

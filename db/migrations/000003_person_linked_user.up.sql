ALTER TABLE person_profiles
ADD COLUMN linked_user_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX person_profiles_linked_user_id_idx
    ON person_profiles(linked_user_id);

CREATE UNIQUE INDEX person_profiles_family_linked_user_uidx
    ON person_profiles(family_id, linked_user_id)
    WHERE linked_user_id IS NOT NULL;

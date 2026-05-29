DROP INDEX IF EXISTS person_profiles_family_linked_user_uidx;
DROP INDEX IF EXISTS person_profiles_linked_user_id_idx;
ALTER TABLE person_profiles DROP COLUMN IF EXISTS linked_user_id;

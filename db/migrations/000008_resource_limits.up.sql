-- NOT VALID preserves existing data while enforcing limits on new writes and
-- updates. Operators can repair legacy violations before VALIDATE CONSTRAINT.
ALTER TABLE users ADD CONSTRAINT limits_users_text CHECK (char_length(email) <= 254 AND char_length(display_name) <= 120 AND char_length(password_hash) <= 512) NOT VALID;
ALTER TABLE families ADD CONSTRAINT limits_families_name CHECK (char_length(name) <= 120) NOT VALID;
ALTER TABLE person_profiles ADD CONSTRAINT limits_people_text CHECK (char_length(name) <= 120 AND char_length(relationship_label) <= 64) NOT VALID;
ALTER TABLE person_income_history ADD CONSTRAINT limits_history_name CHECK (char_length(person_name) <= 120) NOT VALID;
ALTER TABLE housing_groups ADD CONSTRAINT limits_groups_name CHECK (char_length(name) <= 120) NOT VALID;
ALTER TABLE housing_options ADD CONSTRAINT limits_housing_text CHECK (char_length(name) <= 120 AND char_length(location) <= 255 AND char_length(unit_type) <= 120) NOT VALID;
ALTER TABLE housing_options ADD CONSTRAINT limits_housing_overrides CHECK (jsonb_typeof(dia_income_overrides) = 'array' AND jsonb_array_length(dia_income_overrides) <= 50 AND octet_length(dia_income_overrides::text) <= 16384) NOT VALID;
ALTER TABLE sessions ADD CONSTRAINT limits_sessions_hash CHECK (char_length(token_hash) <= 128) NOT VALID;
ALTER TABLE admin_access_keys ADD CONSTRAINT limits_keys_hash CHECK (char_length(token_hash) <= 128) NOT VALID;
ALTER TABLE family_invites ADD CONSTRAINT limits_invites_hash CHECK (char_length(code_hash) <= 128) NOT VALID;

-- Quotas count retained rows, including revoked/used rows. Only deletion or
-- retention releases a slot, so create/revoke loops cannot grow storage forever.
CREATE FUNCTION enforce_user_resource_quota() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE current_count bigint;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.user_id IS DISTINCT FROM OLD.user_id THEN
            RAISE EXCEPTION 'resource owner cannot be changed' USING ERRCODE='23514';
        END IF;
        RETURN NEW;
    END IF;
    PERFORM id FROM users WHERE id=NEW.user_id FOR UPDATE;
    EXECUTE format('SELECT count(*) FROM %I.%I WHERE user_id=$1', TG_TABLE_SCHEMA, TG_TABLE_NAME) INTO current_count USING NEW.user_id;
    IF current_count >= TG_ARGV[0]::int THEN
        RAISE EXCEPTION 'resource quota exceeded: % (maximum % per user)', TG_TABLE_NAME, TG_ARGV[0] USING ERRCODE='P0001', CONSTRAINT='resource_quota';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER sessions_quota BEFORE INSERT OR UPDATE OF user_id ON sessions FOR EACH ROW EXECUTE FUNCTION enforce_user_resource_quota('100');
CREATE TRIGGER admin_access_keys_quota BEFORE INSERT OR UPDATE OF user_id ON admin_access_keys FOR EACH ROW EXECUTE FUNCTION enforce_user_resource_quota('100');

CREATE FUNCTION enforce_family_resource_quota() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE current_count bigint;
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.family_id IS DISTINCT FROM OLD.family_id THEN
            RAISE EXCEPTION 'resource family cannot be changed' USING ERRCODE='23514';
        END IF;
        RETURN NEW;
    END IF;
    PERFORM id FROM families WHERE id=NEW.family_id FOR UPDATE;
    EXECUTE format('SELECT count(*) FROM %I.%I WHERE family_id=$1', TG_TABLE_SCHEMA, TG_TABLE_NAME) INTO current_count USING NEW.family_id;
    IF current_count >= TG_ARGV[0]::int THEN
        RAISE EXCEPTION 'resource quota exceeded: % (maximum % per family)', TG_TABLE_NAME, TG_ARGV[0] USING ERRCODE='P0001', CONSTRAINT='resource_quota';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER person_profiles_quota BEFORE INSERT OR UPDATE OF family_id ON person_profiles FOR EACH ROW EXECUTE FUNCTION enforce_family_resource_quota('20');
CREATE TRIGGER housing_options_quota BEFORE INSERT OR UPDATE OF family_id ON housing_options FOR EACH ROW EXECUTE FUNCTION enforce_family_resource_quota('100');
CREATE TRIGGER housing_groups_quota BEFORE INSERT OR UPDATE OF family_id ON housing_groups FOR EACH ROW EXECUTE FUNCTION enforce_family_resource_quota('50');
-- Retired criteria count too, preserving answers without unbounded soft deletes.
CREATE TRIGGER housing_criteria_quota BEFORE INSERT OR UPDATE OF family_id ON housing_checklist_criteria FOR EACH ROW EXECUTE FUNCTION enforce_family_resource_quota('50');
CREATE TRIGGER family_invites_quota BEFORE INSERT OR UPDATE OF family_id ON family_invites FOR EACH ROW EXECUTE FUNCTION enforce_family_resource_quota('100');
CREATE TRIGGER income_history_quota BEFORE INSERT OR UPDATE OF family_id ON person_income_history FOR EACH ROW EXECUTE FUNCTION enforce_family_resource_quota('5000');

CREATE FUNCTION enforce_membership_quota() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        IF NEW.family_id IS DISTINCT FROM OLD.family_id OR NEW.user_id IS DISTINCT FROM OLD.user_id THEN
            RAISE EXCEPTION 'membership identity cannot be changed' USING ERRCODE='23514';
        END IF;
        RETURN NEW;
    END IF;
    -- Match JoinFamilyByCode's lock order. Repeated joins remain idempotent.
    PERFORM id FROM families WHERE id=NEW.family_id FOR UPDATE;
    PERFORM id FROM users WHERE id=NEW.user_id FOR UPDATE;
    IF EXISTS (SELECT 1 FROM family_members WHERE family_id=NEW.family_id AND user_id=NEW.user_id) THEN RETURN NEW; END IF;
    IF (SELECT count(*) FROM family_members WHERE user_id=NEW.user_id) >= 10 THEN
        RAISE EXCEPTION 'resource quota exceeded: maximum 10 families per user' USING ERRCODE='P0001', CONSTRAINT='resource_quota';
    END IF;
    IF (SELECT count(*) FROM family_members WHERE family_id=NEW.family_id) >= 20 THEN
        RAISE EXCEPTION 'resource quota exceeded: maximum 20 members per family' USING ERRCODE='P0001', CONSTRAINT='resource_quota';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER family_members_quota BEFORE INSERT OR UPDATE OF family_id,user_id ON family_members FOR EACH ROW EXECUTE FUNCTION enforce_membership_quota();

CREATE INDEX users_page_idx ON users(created_at DESC,id DESC);
CREATE INDEX families_page_idx ON families(created_at DESC,id DESC);
CREATE INDEX members_page_idx ON family_members(family_id,joined_at,user_id);
CREATE INDEX people_page_idx ON person_profiles(family_id,created_at DESC,id DESC);
CREATE INDEX housing_page_idx ON housing_options(family_id,created_at DESC,id DESC);
CREATE INDEX groups_page_idx ON housing_groups(family_id,created_at DESC,id DESC);
CREATE INDEX history_page_idx ON person_income_history(family_id,recorded_at,id);
CREATE INDEX invites_family_idx ON family_invites(family_id);
CREATE INDEX criteria_family_idx ON housing_checklist_criteria(family_id);
CREATE INDEX sessions_expiry_idx ON sessions(expires_at);
CREATE INDEX sessions_revocation_idx ON sessions(revoked_at) WHERE revoked_at IS NOT NULL;
CREATE INDEX keys_expiry_idx ON admin_access_keys(expires_at);
CREATE INDEX keys_used_idx ON admin_access_keys(used_at) WHERE used_at IS NOT NULL;
CREATE INDEX invites_expiry_idx ON family_invites(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX invites_revocation_idx ON family_invites(revoked_at) WHERE revoked_at IS NOT NULL;

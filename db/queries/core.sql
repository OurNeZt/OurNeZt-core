-- name: CreateUser :one
INSERT INTO users (email, display_name, role, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: DisableUser :exec
UPDATE users
SET disabled_at = now(), updated_at = now()
WHERE id = $1;

-- name: CreateFamily :one
INSERT INTO families (name, family_type)
VALUES ($1, $2)
RETURNING *;

-- name: AddFamilyMember :exec
INSERT INTO family_members (family_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (family_id, user_id)
DO UPDATE SET role = EXCLUDED.role;

-- name: ListUserFamilies :many
SELECT f.*
FROM families f
JOIN family_members fm ON fm.family_id = f.id
WHERE fm.user_id = $1
ORDER BY f.created_at DESC;

-- name: ListPersonProfilesByFamily :many
SELECT * FROM person_profiles
WHERE family_id = $1
ORDER BY created_at DESC;

-- name: ListHousingOptionsByFamily :many
SELECT * FROM housing_options
WHERE family_id = $1
ORDER BY created_at DESC;


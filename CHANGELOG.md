# Change Log
All notable changes to this project will be documented in this file.
 
The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

---

## [v1.3.3] - 2026-06-15

### Added
- (fill)

### Changed
- (fill)

### Fixed
- (fill)

### Removed
- (fill)


## [v1.3.2] - 2026-06-01

### Added
- Optional gRPC server TLS support via:
    - `GRPC_TLS_CERT_FILE`
    - `GRPC_TLS_KEY_FILE`
- Startup logging now indicates whether gRPC TLS is enabled.

### Changed
- gRPC server bootstrap now conditionally applies TLS server credentials when cert/key are configured.

### Fixed
- Household gross income aggregation now consistently includes stored gross values across profile statuses.
- CPF handling aligned to planning rules for selected statuses (student/NSF/part-time no employee CPF deduction in summary flow).

### Removed
- No removals in this release.


## [v1.3.1] - 2026-06-01

### Added
- Explicit CPF exclusion handling for `student` and `full_time_nsf` employment statuses in income processing rules.

### Changed
- Gross income calculation flow was corrected to use the proper values/order of operations before affordability and CPF computations.
- CPF contribution logic now applies only to CPF-eligible employment categories.

### Fixed
- Fixed gross income calculation issue that caused inaccurate totals in financial outputs.
- Fixed CPF computation behavior so student and NSF gross income are excluded from CPF calculations.

### Removed
- No removals in this release.

## [v1.3.0] - 2026-05-31

### Added
- Per-housing DIA income override support so each user can set projected future income specific to a housing option.
- Storage/model support for housing-scoped DIA income inputs (`dia_income_overrides`) in backend data layer.

### Changed
- DIA affordability calculations now read housing-specific projected income values instead of using one shared value across all housing options.
- User-edit flow now correctly persists profile updates through to the database layer.

### Fixed
- Fixed issue where user profile edits appeared to save but were not properly written to the database.
- Fixed cross-housing projection sync issue where editing future income for one housing option unintentionally affected other housing options.

### Removed
- No removals in this release.

## [v1.2.0] - 2026-05-29

### Added
- Admin bootstrap via environment variables (`BOOTSTRAP_ADMIN_EMAIL`, `BOOTSTRAP_ADMIN_PASSWORD`, `BOOTSTRAP_ADMIN_DISPLAY_NAME`) for first-deploy setup.
- One-time admin access key flow for gated admin dashboard access.
- Admin-only user listing endpoint to support dashboard user tables (including user IDs and account status).
- `must_change_password` enforcement for bootstrap/admin first login hardening.

### Changed
- Bootstrap behavior now auto-creates or promotes the first admin account when no active admin exists.
- User/session admin flows updated so disabling a user also revokes active sessions.
- Person profile data model expanded to support linked/editable user relationships (`linked_user_id`).

### Fixed
- Docker startup failure fixed (`exec /app/docker-entrypoint.sh: no such file or directory`) by normalizing entrypoint script to LF and enforcing line endings with `.gitattributes`.
- SQLC generated files synchronized with schema/query updates to prevent CI drift failures.
- Transaction rollback lint issue fixed (`errcheck` on deferred rollback).

### Removed
- No removals in this release.

## [v1.1.2] - 2026-05-28

### Added
- Add protobuf function for users details

### Changed
- Refactor server to return user details

### Fixed
- No fixes in this release.

### Removed
- No removals in this release.

## [v1.1.1] - 2026-05-26

### Added
- No Additions in this release.

### Changed
- No Changes in this release.

### Fixed
- Fix wrongly synced files

### Removed
- No removals in this release.

## [v1.1.0] - 2026-05-26

### Added
- Added admin bootstrap support for creating the initial admin user through environment-based configuration.
- Added deployment-friendly first-admin initialisation flow for fresh OurNeZt Core instances.
- Added configuration support for admin bootstrap credentials and startup validation.

### Changed
- Refactored auth server startup flow for cleaner initialisation and dependency handling.
- Updated auth context and handler logic to better support the new authentication flow.
- Updated the auth API contract to match the revised authentication schema.
- Regenerated Go protobuf bindings for the updated auth API.
- Updated domain models and user repository logic to support the admin bootstrap flow.
- Updated database migrations for the revised auth and user model requirements.

### Fixed
- Fixed first-deployment setup by removing the need to manually insert the first admin user into the database.
- Fixed auth flow consistency between startup configuration, API handlers, domain models, and repository logic.

### Removed
- Removed reliance on manual database insertion for initial admin account creation.

## [v1.0.0] - 2026-05-17

### Added
- Added the initial OurNeZt Core backend API foundation.
- Added gRPC services for authentication, family management, person profiles, finance calculations, housing planning, and dashboard aggregation.
- Added PostgreSQL persistence with database migrations, repository interfaces, and PostgreSQL-backed repository implementations.
- Added user registration, login, session validation, and user disabling support.
- Added session-token based authentication using gRPC metadata.
- Added role-based access control for protected backend operations.
- Added family and household management, including family creation, member listing, invite code generation, and joining by invite code.
- Added person profile management for household members, including income, savings, CPF, employment, and expense fields.
- Added household finance calculations for income, CPF contributions, take-home income, expenses, savings, and projected surplus.
- Added housing option management for BTO, resale, EC, private, and other housing planning scenarios.
- Added housing affordability calculations, including upfront cost, loan amount, monthly mortgage, CPF usage, cash top-up, and affordability rating.
- Added dashboard aggregation to combine household profiles, finance summaries, and housing affordability results.
- Added Docker support for containerised deployment.
- Added GitHub Actions workflows for CI, release preparation, tagging, GitHub Releases, and GHCR Docker image publishing.
- Added project documentation, including README, release flow, architecture overview, logo, and architecture diagram.

### Changed
- Updated backend authorization flow to resolve actor identity from session metadata instead of trusting request-provided user IDs.
- Updated release workflow so release tags are created after the release branch is merged into `stable`.
- Updated prepare-release workflow to prepare changelog entries and release branches without creating tags early.
- Updated repository and server layers to enforce family role checks for create, update, and delete operations.

### Fixed
- Fixed insecure user and actor handling by protecting admin user creation, user disabling, and family write operations.
- Fixed PostgreSQL error mapping for common constraint and validation errors.
- Fixed test coverage for updated authorization and repository interface changes.
- Fixed release workflow behaviour to avoid skipped releases caused by tags being created before the final release merge.

### Removed
- Removed early tag creation from the prepare-release workflow.
- Removed reliance on request-provided actor IDs for authenticated backend operations.

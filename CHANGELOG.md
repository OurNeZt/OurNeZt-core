# Change Log
All notable changes to this project will be documented in this file.
 
The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

---

## [v2.0.0] - 2026-11-13

### Added

- `RevokeCurrentSession` RPC for idempotent revocation of the session token supplied in request metadata.
- Cursor pagination for user, family, family member, person profile, income history, housing option, and housing group lists, with `page_size`, `page_token`, and `next_page_token` fields.
- Database-enforced resource quotas and persisted input limits through migration `000008_resource_limits`, including indexes for paginated queries.
- Bounded credential cleanup at startup and every 10 minutes, deleting up to 500 rows per credential table per pass after the seven-day retention period.
- Configurable gRPC TLS with `GRPC_TLS_CERT_FILE` and `GRPC_TLS_KEY_FILE`, plus production startup validation.
- An embedded offline common-password blocklist and a shared policy for user/admin creation, password changes, and bootstrap provisioning.
- Regression coverage and reviewer checks for session revocation, concurrent invite rotation, login enumeration, password policy, resource quotas, pagination, retention, and TLS.

### Changed

- List APIs now default to 50 records and accept explicit page sizes from 1 to 100. Clients must follow `next_page_token` to retrieve complete collections; existing clients no longer receive every row in one response.
- Production startup now requires gRPC TLS unless `GRPC_ALLOW_INSECURE=true` explicitly permits a trusted plaintext hop. Partial or invalid TLS configuration does not fall back to plaintext.
- New passwords must contain at least 15 Unicode code points, fit within 1,024 UTF-8 bytes, contain no whitespace, and avoid the common-password blocklist. Existing passwords remain usable for login and current-password verification within the input limits; no forced reset is introduced.
- Successful password changes atomically revoke every session for the user, including the current session, and require a fresh login. Bootstrap credential replacement also revokes existing sessions.
- Incoming gRPC messages are capped at 1 MiB, with field and repeated-input limits checked before expensive work. Persisted strings and resource counts are bounded by database checks and quotas.
- Families can hold up to 100 housing options and 50 checklist criteria; checklist scoring supports the full 5,000-answer matrix. Other per-user and per-family quotas are documented in the resource limits guide.
- Internal dashboard and grant calculations read complete bounded collections or fail explicitly when legacy data exceeds their aggregate limits. Housing list summaries load answers only for the selected page.

### Fixed

- Prevented revoked logout tokens and sessions invalidated by password changes from remaining usable, including races between login and password updates.
- Made family invite regeneration atomically revoke all earlier active codes, with locking to coordinate concurrent regenerations and joins and rollback on failure.
- Unified login failures for unknown accounts, incorrect passwords, disabled accounts, lookup failures, and malformed password hashes. Dummy Argon2id verification reduces account-dependent timing differences while preserving rate and concurrency limits.
- Rejected invalid new passwords before hashing or account mutation while preserving the exact submitted credential bytes and existing sessions after failed changes.
- Prevented concurrent requests from exceeding resource quotas or leaving partial family, invite, profile, or income-history updates after a failure.
- Made GitHub Actions cache export best-effort so cache reservation failures do not fail a release after its image is built and pushed. Build and image push failures still fail the workflow.

### Removed

- Unbounded list responses from the paginated APIs.

### Upgrade Notes

- Apply migration `000008_resource_limits` before starting Core 2.0. Audit oversized legacy fields and over-quota collections, repair them explicitly, and validate the new length/JSON constraints after cleanup. The migration preserves existing records while enforcing limits on new writes and updates.
- Update client bindings and pagination handling before rollout where collections can exceed 50 records. The matching Web 2.0 release does not yet follow pagination tokens, so later records can be omitted from its lists and housing comparisons.
- Configure production gRPC certificates or explicitly allow a trusted internal plaintext hop. Web enforces its own transport configuration separately; browser-facing HTTPS does not encrypt the Web-to-Core connection.
- Deploy Core before the matching Web release to provide the new session revocation RPC. Clients must require a fresh login after successful password changes.
- Replace weak bootstrap credentials before creating or promoting an admin. Existing credentials are not automatically reset, and invalid replacement passwords leave accounts and sessions unchanged.
- Credential retention permanently removes eligible expired, revoked, or used rows after seven days; rolling back the migration cannot restore rows already deleted. Financial history is retained rather than automatically pruned.

## [v1.6.0] - 2026-10-09

### Added

- Family-level housing checklists with reusable criteria, display ordering, optional weights, and criterion retirement.
- Per-housing checklist answers with completion states, ratings, and criterion-specific notes.
- Weighted housing evaluation scores and completion summaries in housing API responses.
- Authenticated gRPC APIs for checklist management and housing evaluations with family authorization checks.
- Shared plain-text housing notes with a 10,000-character limit and a dedicated update API.
- Database migrations `000006_housing_checklist` and `000007_housing_notes`.
- Configurable authentication limits by account and transport peer IP.
- Shared concurrency limits for password verification and hashing.
- Retry-delay information in throttled gRPC responses.
- Regression tests for checklist scoring, housing notes, authentication throttling, and user-creation authorization.
- PostgreSQL integration test coverage in CI.

### Changed

- Housing API contracts and generated bindings now include checklist evaluations and shared notes.
- All `CreateUser` requests now require an authenticated, active admin session.
- Authentication requests are throttled before expensive password work.
- Authentication failures now use generic credential responses without exposing account status.
- Authentication limits recover automatically as rate windows expire and active password operations finish.

### Fixed

- Prevented unauthenticated and non-admin callers from creating ordinary users directly through Core.
- Prevented rejected user-creation requests from reaching password hashing or consuming the account creation budget.
- Preserved shared housing notes when housing details, group assignments, or visibility settings are updated.
- Corrected missing-session handling to return `Unauthenticated`.

### Removed

- No removals in this release.

## [v1.5.0] - 2026-08-04

### Added
- Housing group support with family-scoped `housing_groups` persistence.
- Optional housing group assignment for each housing option.
- Per-housing-option dashboard visibility control.
- gRPC APIs for creating, listing, renaming, and deleting housing groups.
- gRPC APIs for assigning housing options to groups and updating visibility.
- Group-level bulk visibility update support for housing options.
- Database migration for housing groups and dashboard visibility fields.

### Changed
- Household dashboard generation now includes only housing options marked as visible on the dashboard.
- Housing service contracts and mappings now include housing group and dashboard visibility fields.
- Housing repository flow now validates group assignment within the same family.

### Fixed
- Prevented group-level visibility actions from acting like a permanent parent restriction by implementing them as bulk updates only.
- Ensured deleting a housing group keeps housing options intact and safely returns them to an ungrouped state.

### Removed
- No removals in this release.

## [v1.4.0] - 2026-06-26

### Added
- Added housing grant calculation support for HDB-related housing options.
- Added staged downpayment support to housing affordability calculations.
- Added backend validation for housing-type-specific calculation fields.
- Added tests for updated housing grant, loan tenure, and housing validation logic.

### Changed
- Updated loan tenure calculations for HDB/BTO and private property rules.
- Updated housing grant income calculation to use the correct monthly average income table.
- Updated grant-to-average calculation logic.
- Updated housing affordability calculation logic for staged downpayment scenarios.
- Updated housing protobuf contracts and regenerated generated code for housing calculation changes.
- Updated server housing mappings for the revised housing calculation flow.
- Refactored housing calculation logic for resale HDB, EC, private condo, landed, and other property types.

### Fixed
- Fixed non-HDB housing options accepting HDB-only calculation fields.
- Fixed condo, landed, and other non-HDB housing validation against HDB-only loan/grant fields.
- Fixed unused housing calculation variables after housing-type-specific logic changes.

### Removed
- Removed unused housing calculation variables for property types where they no longer apply.

## [v1.3.3] - 2026-06-15

### Added
- Added contextual help tooltips across key housing inputs, calculated values, and planning views
- Added centralized housing tooltip content management for easier updates and reuse
- Added official reference links in selected tooltips for buyer stamp duty, HDB grant information, and HDB interest rate details
- Added clearer upfront payment visibility in housing detail, comparison, and dashboard timeline views

### Changed
- Changed loan tenure input in the housing form from months to years
- Changed `BTO + HDB loan` tenure handling to auto-assume 25 years as a system-derived value
- Changed selected housing form fields to display system-derived values using placeholder/read-only behavior instead of looking user-entered
- Changed the first housing payment checkpoint wording from `Initial Downpayment` to `Total Initial Payment`
- Changed the Payment vs Income Timeline chart labels, legends, and stage naming for better readability
- Changed tooltip styling and interaction behavior to be more minimal, consistent, and mobile-friendly

### Fixed
- Fixed tooltip positioning, clipping, visibility, and interaction issues across housing pages
- Fixed “See more” tooltip links so they remain usable and clickable
- Fixed housing grant display behavior to better reflect system-derived calculation handling
- Fixed initial payment checkpoint calculation/display alignment for housing planning
- Fixed the total initial payment display to include buyer stamp duty while keeping later downpayment stages separate
- Fixed submission behavior for auto-derived `BTO + HDB loan` tenure values
- Fixed housing inputs that were showing placeholder-like default values as actual entered values

### Removed
- Removed misleading prefilled-looking default values from selected housing inputs
- Removed inconsistent tooltip presentation across different housing sections

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

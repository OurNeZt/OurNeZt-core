# Change Log
All notable changes to this project will be documented in this file.
 
The format is based on [Keep a Changelog](http://keepachangelog.com/)
and this project adheres to [Semantic Versioning](http://semver.org/).

---

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

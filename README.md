<a id="readme-top"></a>

<!-- PROJECT SHIELDS -->
<p align="center">
  <a href="https://github.com/OurNeZt/OurNeZt-core/actions/workflows/ci.yaml">
    <img src="https://img.shields.io/github/actions/workflow/status/OurNeZt/OurNeZt-core/ci.yaml?branch=dev&style=for-the-badge&label=CI" alt="CI Status">
  </a>
  <a href="https://github.com/OurNeZt/OurNeZt-core/releases">
    <img src="https://img.shields.io/github/v/release/OurNeZt/OurNeZt-core?style=for-the-badge" alt="Latest Release">
  </a>
  <a href="https://github.com/OurNeZt/OurNeZt-core/blob/stable/LICENSE">
    <img src="https://img.shields.io/github/license/OurNeZt/OurNeZt-core?style=for-the-badge" alt="License">
  </a>
  <a href="https://github.com/OurNeZt/OurNeZt-core/pkgs/container/ournezt-core">
    <img src="https://img.shields.io/badge/GHCR-ournezt--core-blue?style=for-the-badge&logo=github" alt="GHCR Image">
  </a>
</p>

<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/OurNeZt/OurNeZt-core">
    <img src="assets/OurNeZt-Core-Logo.png" alt="OurNeZt Core Logo" width="450">
  </a>

  <h1 align="center">OurNeZt Core</h1>

  <p align="center">
    Backend API service for household finance, housing planning, and shared OurNeZt domain logic.
    <br />
    <br />
    <a href="https://github.com/OurNeZt/OurNeZt-core/issues/new?labels=bug&template=bug-report.md">Report Bug</a>
    &middot;
    <a href="https://github.com/OurNeZt/OurNeZt-core/issues/new?labels=enhancement&template=feature-request.md">Request Feature</a>
    &middot;
    <a href="https://github.com/OurNeZt/OurNeZt-core/releases">Releases</a>
  </p>
</div>

---

<!-- TABLE OF CONTENTS -->
<details>
  <summary>Table of Contents</summary>
  <ol>
    <li>
      <a href="#about-the-project">About The Project</a>
      <ul>
        <li><a href="#what-it-does">What It Does</a></li>
        <li><a href="#built-with">Built With</a></li>
      </ul>
    </li>
    <li><a href="#architecture-overview">Architecture Overview</a></li>
    <li>
      <a href="#getting-started">Getting Started</a>
      <ul>
        <li><a href="#prerequisites">Prerequisites</a></li>
        <li><a href="#installation">Installation</a></li>
        <li><a href="#configuration">Configuration</a></li>
      </ul>
    </li>
    <li><a href="#login-response-protections">Login Response Protections</a></li>
    <li><a href="#session-revocation">Session Revocation</a></li>
    <li><a href="#family-invite-codes">Family Invite Codes</a></li>
    <li><a href="#running-the-service">Running The Service</a></li>
    <li><a href="#testing">Testing</a></li>
    <li><a href="#docker">Docker</a></li>
    <li><a href="#release-flow">Release Flow</a></li>
    <li><a href="#roadmap">Roadmap</a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#contact">Contact</a></li>
  </ol>
</details>

---

## About The Project

**OurNeZt Core** is the backend API service for the OurNeZt ecosystem. It provides the shared backend foundation for household finance planning, family management, person profiles, income calculations, housing affordability checks, and dashboard aggregation.

This repository is designed to act as the core backend module used by future OurNeZt applications, such as the web app, admin tools, or other internal services.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## What It Does

OurNeZt Core currently provides backend support for:

- User registration, login, session validation, and user disabling.
- Session-token based authentication using gRPC metadata.
- Current-session revocation on logout and all-session revocation after password changes.
- Family and household creation, membership, invite codes, and role-based access control.
- Person profile management for household members.
- Income, CPF, cash savings, expense, and household surplus calculations.
- Housing option management for BTO, resale, EC, private, and other housing types.
- Housing affordability calculations based on household financial data.
- Dashboard aggregation for household-level summaries.

The service is built as a backend-first gRPC API, making it suitable for frontend clients, API gateways, and other internal services.

---

## Built With

<p align="left">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/gRPC-244c5a?style=for-the-badge&logo=grpc&logoColor=white" alt="gRPC">
  <img src="https://img.shields.io/badge/Protocol%20Buffers-4285F4?style=for-the-badge&logo=google&logoColor=white" alt="Protocol Buffers">
  <img src="https://img.shields.io/badge/PostgreSQL-336791?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/sqlc-000000?style=for-the-badge" alt="sqlc">
  <img src="https://img.shields.io/badge/Buf-000000?style=for-the-badge" alt="Buf">
  <img src="https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/GitHub%20Actions-2088FF?style=for-the-badge&logo=github-actions&logoColor=white" alt="GitHub Actions">
</p>

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Architecture Overview

<p align="center">
  <img src="assets/OurNeZt-core-arch-diag-dark.png" alt="OurNeZt Core Architecture Diagram" width="100%">
</p>

OurNeZt Core is designed as the backend API layer for the OurNeZt ecosystem. Client-facing applications such as the web app, client app, or an API gateway communicate with the core service through gRPC or HTTP/REST through a gateway layer.

At the centre of the architecture is the **OurNeZt Core gRPC API**, which routes requests into the internal service layer. The backend is split into focused service modules:

- **Auth Service** handles user authentication, login sessions, and account access control.
- **Family Service** manages households, family membership, invite codes, and role-based access.
- **Person Service** manages household member profiles, income details, CPF values, and savings data.
- **Finance Service** performs household income, CPF, expense, and surplus calculations.
- **Housing Service** manages housing options and affordability-related inputs.
- **Dashboard Service** aggregates family, finance, person, and housing data into dashboard-ready summaries.

PostgreSQL is used as the persistence layer for users, sessions, families, person profiles, housing options, and related financial planning data.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Getting Started

### Prerequisites

Install the following tools:

- Go
- PostgreSQL
- Docker
- Buf
- sqlc

Optional but recommended:

- `make`
- `grpcurl`
- Docker Compose

---

### Installation

Clone the repository:

```bash
git clone git@github.com:OurNeZt/OurNeZt-core.git
cd OurNeZt-core
```

Install dependencies:

```bash
go mod download
```

Run tests:

```bash
go test ./...
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Configuration

The service is configured using environment variables.

Example:

```bash
export GRPC_ADDR=":50051"
export DATABASE_URL="postgres://ournezt:ournezt@localhost:5432/ournezt?sslmode=disable"
export SESSION_TOKEN_BYTES="32"
export BOOTSTRAP_ADMIN_EMAIL = "admin@example.com"
export BOOTSTRAP_ADMIN_PASSWORD = "P@ssw0rd"
export BOOTSTRAP_ADMIN_DISPLAY_NAME = "Admin"
```

Behaviour:

- If no active admin exists, startup creates (or promotes) this account to admin.
- The bootstrap admin is forced to change password on first login.
- If any active admin already exists, bootstrap is skipped.

---

## Login Response Protections

Login rejects unknown emails, wrong passwords, disabled accounts, account lookup failures, and malformed stored password hashes with the same gRPC `Unauthenticated` response: `Invalid email or password.` Failed session creation also uses this generic response. Request cancellation, deadlines, and authentication traffic limits remain distinct.

Unknown accounts and lookup failures perform dummy Argon2id verification using the configured password cost. Disabled accounts undergo password verification before rejection, and malformed stored hashes trigger dummy verification. All this work runs within the existing account rate and password concurrency limits. This reduces timing differences; it does not guarantee identical latency, especially if stored hashes use different Argon2 settings.

Run the account enumeration regression checks without a database:

```bash
go test ./internal/platform/security ./internal/service ./internal/server -run 'TestDummyPasswordHash|TestVerifyPasswordRejectsInvalidStoredParameters|TestLogin' -count=1 -v
```

The checks cover identical gRPC codes, messages, and empty error details; failed logins creating no session; configured dummy verification cost; verification for disabled users and lookup failures; and rate/concurrency limits during dummy work. `TestLoginGRPCFailureResponses` starts the real Auth server and service on an in-memory gRPC transport, checks the client-visible responses for unknown emails, wrong passwords, disabled users with both correct and wrong passwords, lookup failures, and malformed hashes, then verifies a successful login still creates a session. No running Core instance or PostgreSQL is needed. The service tests inspect the verification work rather than asserting noisy timing thresholds.

For a running Web + Core deployment, use Web's `cmd/login-enumeration-check` with existing active and disabled test fixture emails. Its README includes the reviewer command and expected results.

---

## Session Revocation

Sessions use opaque tokens; Core stores their hashes and checks expiry and `revoked_at` on validation and authenticated RPCs.

- `RevokeCurrentSession` revokes only the token presented through `x-session-token` or Bearer metadata. Its request has no user ID or target token field. Unknown, expired, and already-revoked tokens are safe to revoke repeatedly; missing session metadata is rejected.
- A successful `ChangePassword` atomically updates the password, clears the first-login password-change flag, and revokes every session for that user, including the current one. Clients must require a fresh login. Failed password checks or failed database operations leave credentials and sessions intact.
- Session creation locks the user row and compares the password hash verified during login. A login that verified old credentials cannot create a surviving session after a password change. Password updates also compare the verified hash to prevent stale concurrent updates.
- Bootstrap credential replacement revokes existing sessions in the same transaction.

No new database migration is required: the existing session revocation column is used. Deploy the updated Core before the matching Web release. Web retains its cookie and offers retry when logout revocation cannot be confirmed, including when an older Core lacks the new RPC.

This follows the [OWASP session management guidance](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html) on server-side logout invalidation and reauthentication after password changes.

---

## Family Invite Codes

Regenerating a family's invite code revokes every previous unrevoked code for that family and creates the replacement in one transaction. Only the replacement can be used afterward. This also handles older data containing several active codes, including codes without an expiry. Other families' codes and existing memberships are unaffected.

Only family owners and family admins can regenerate codes. If authorization, revocation, or replacement creation fails, the transaction rolls back and the previous code remains usable until its original expiry. Codes continue to be stored as hashes; the raw replacement is returned only after a successful commit.

Regeneration and joining coordinate through a lock on the family. Simultaneous regenerations leave only the last replacement active, including when no code existed initially. A join that reaches the lock after regeneration must recheck revocation and expiry before adding the member. A join that holds the lock first completes before regeneration; rotating codes does not remove members who already joined.

No database migration, new environment setting, or Web change is required: the existing `family_invites.revoked_at` column is used. Existing codes are revoked when that family next regenerates its code.

---

## Running The Service

Run locally:

```bash
go run ./cmd/core
```

The gRPC server should start using the configured bind address.

Example using `grpcurl`:

```bash
grpcurl -plaintext localhost:50051 list
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Testing

Run all tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run tests with coverage:

```bash
go test -v -cover ./...
```

### Session Revocation Tests

Run the token-reuse and gRPC lifecycle regressions without a database:

```bash
go test ./internal/server ./internal/service -run 'TestSession|TestChangePassword' -count=1 -v
```

For the PostgreSQL transaction and concurrency checks, set `TEST_DATABASE_URL` to a disposable test database:

```bash
export TEST_DATABASE_URL='postgres://postgres:test-password@localhost:55432/ournezt_test?sslmode=disable'
go test ./internal/repository/postgres -run TestSessionRevocationPostgres -count=1 -v
```

Each test creates an isolated schema, applies the migrations, and removes that schema afterward. The checks cover token scope and expiry, rollback when revocation fails, stale credential rejection, and both orderings of concurrent login/password changes. Without `TEST_DATABASE_URL`, database tests are skipped. The existing CI PostgreSQL service runs them automatically with `go test ./...`.

### Family Invite Revocation Tests

For a readable reviewer report, run this from the Core repository root with `TEST_DATABASE_URL` configured:

```bash
go run ./cmd/family-invite-check
```

The command runs the actual PostgreSQL regression tests and prints one plain-language result per scenario, such as `PASS: Failed replacements leave the previous invite usable.` It verifies all eight required scenarios completed and exits nonzero for any failure, skipped check, missing result, timeout, or test-process error. Raw test output is hidden; the command gives the focused `go test` command when technical details are needed.

It does not need a running Core server, account credentials, or an existing family. Tests use temporary schemas and clean them up afterward. No existing families are changed. This verifies the checked-out code against PostgreSQL; it does not probe an already-deployed Core server.

For a self-contained local review with Docker and PowerShell:

```powershell
docker run --detach --rm --name ournezt-invite-review -e POSTGRES_PASSWORD=invite-review-test -e POSTGRES_DB=ournezt_test -p 127.0.0.1:15455:5432 postgres:17-bookworm
# Wait until this reports that PostgreSQL is accepting connections:
docker exec ournezt-invite-review pg_isready -U postgres
$env:TEST_DATABASE_URL = 'postgres://postgres:invite-review-test@127.0.0.1:15455/ournezt_test?sslmode=disable'
go run ./cmd/family-invite-check
docker stop ournezt-invite-review
```

Port 15455 must be available. For an existing disposable test database, set `TEST_DATABASE_URL` directly and omit the Docker commands. The report command refuses to run without a database URL, so skipped integration tests cannot look like a successful review. Use `-timeout 5m` if the default two-minute test budget is insufficient.

For detailed developer output, run the focused tests directly:

Set `TEST_DATABASE_URL` to a disposable PostgreSQL database. Each test creates an isolated schema, applies the migrations, and removes that schema afterward. The existing CI PostgreSQL service runs these tests automatically.

```bash
export TEST_DATABASE_URL='postgres://postgres:test-password@localhost:15455/ournezt_test?sslmode=disable'
go test ./internal/repository/postgres -run TestFamilyInvitePostgres -count=1 -v
```

In PowerShell, set the variable with `$env:TEST_DATABASE_URL = 'postgres://postgres:test-password@localhost:15455/ournezt_test?sslmode=disable'`, then run the same `go test` command. Without this variable, the database checks explicitly report `SKIP`.

The tests check replaced and expired codes, multiple legacy codes, owner/admin permissions, unrelated families, repeated joins, rollback failures, simultaneous regenerations, both orderings of joining and regeneration, and expiry while a join waits for a lock.

For a manual review with separate owner and joining accounts:

| Check | Expected result |
| --- | --- |
| Generate code A, regenerate to code B, then join using A | Rejected; no new membership |
| Join using code B | Accepted |
| Regenerate again to code C, then join using B | Rejected; code C works |
| Try a code for a different family | That family's code remains usable |
| Attempt regeneration as a member or viewer | Rejected; the current code remains usable |

Optional developer code coverage report (shows which source code ran; this is not a feature pass/fail report):

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Docker

Build the image locally:

```bash
docker build -t ournezt-core:local .
```

Run the container:

```bash
docker run --rm \
  -p 50051:50051 \
  -e GRPC_ADDR=":50051" \
  -e DATABASE_URL="postgres://ournezt:ournezt@host.docker.internal:5432/ournezt?sslmode=disable" \
  -e RUN_MIGRATIONS="true" \
  -e BOOTSTRAP_ADMIN_EMAIL="admin@ournezt.local" \
  -e BOOTSTRAP_ADMIN_PASSWORD="P@ssw0rd" \
  -e BOOTSTRAP_ADMIN_DISPLAY_NAME="Admin" \
  ournezt-core:local
```

Container startup behavior:

- Runs migrations automatically before starting the app (`RUN_MIGRATIONS=true` by default).
- Uses `MIGRATIONS_PATH` if you need a custom migrations directory.
- Then starts the gRPC server, where bootstrap admin logic can run from `BOOTSTRAP_ADMIN_*` env vars.

Released images are published to GitHub Container Registry:

```bash
docker pull ghcr.io/OurNeZt/ournezt-core:latest
```

or for a specific release:

```bash
docker pull ghcr.io/OurNeZt/ournezt-core:vX.Y.Z
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Release Flow

This repository uses a controlled release flow:

```text
dev -> release/vX.Y.Z -> stable
```

The release process is:

1. Run the `prepare-release` workflow manually.
2. Select the version bump type: `patch`, `minor`, or `major`.
3. The workflow creates a `release/vX.Y.Z` branch and updates `CHANGELOG.md`.
4. Review and update the generated changelog entry.
5. Merge the release PR into `stable`.
6. The `release` workflow creates the Git tag, GitHub Release, and Docker images.

Docker images are tagged as:

```text
ghcr.io/OurNeZt/ournezt-core:vX.Y.Z
ghcr.io/OurNeZt/ournezt-core:latest
```

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Roadmap

- [x] Backend API foundation
- [x] gRPC service definitions
- [x] PostgreSQL repository layer
- [x] Authentication and session handling
- [x] Logout and password-change session revocation
- [x] Family and household management
- [x] Person profile management
- [x] Finance and CPF calculation logic
- [x] Housing affordability calculation logic
- [x] Dashboard aggregation
- [x] Docker image build and release workflow
- [ ] API gateway integration
- [ ] Frontend integration
- [ ] More complete end-to-end auth tests
- [ ] Expanded deployment manifests
- [ ] Observability, metrics, and structured logging

See the [open issues](https://github.com/OurNeZt/OurNeZt-core/issues) for planned improvements and known issues.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Contributing

Contributions are welcome.

For normal development:

1. Create a feature branch from `dev`.

   ```bash
   git checkout dev
   git pull
   git checkout -b feature/your-feature-name
   ```

2. Make your changes.

3. Run tests.

   ```bash
   go test ./...
   ```

4. Commit your changes using a conventional commit style.

   ```bash
   git commit -m "feat: add new backend capability"
   ```

5. Open a pull request into `dev`.

For release preparation, use the release workflow instead of manually creating tags.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## License

Distributed under the Apache License 2.0. See `LICENSE` for more information.

<p align="right">(<a href="#readme-top">back to top</a>)</p>

---

## Contact

Project Organisation: [OurNeZt](https://github.com/OurNeZt)

Repository: [https://github.com/OurNeZt/OurNeZt-core](https://github.com/OurNeZt/OurNeZt-core)

<p align="right">(<a href="#readme-top">back to top</a>)</p>

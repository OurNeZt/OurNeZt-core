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
export OURNEZT_GRPC_ADDR=":50051"
export OURNEZT_DATABASE_URL="postgres://ournezt:ournezt@localhost:5432/ournezt?sslmode=disable"
export OURNEZT_SESSION_TOKEN_BYTES="32"
```

Adjust the variable names based on the current config package if they change.

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

For authenticated requests, pass the session token as metadata:

```bash
grpcurl \
  -plaintext \
  -H "authorization: Bearer <session-token>" \
  localhost:50051 \
  ournezt.v1.AuthService/ValidateSession
```

Alternatively:

```bash
grpcurl \
  -plaintext \
  -H "x-session-token: <session-token>" \
  localhost:50051 \
  ournezt.v1.AuthService/ValidateSession
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

Generate an HTML coverage report:

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
  -e OURNEZT_GRPC_ADDR=":50051" \
  -e OURNEZT_DATABASE_URL="postgres://ournezt:ournezt@host.docker.internal:5432/ournezt?sslmode=disable" \
  ournezt-core:local
```

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

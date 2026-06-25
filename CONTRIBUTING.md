# Contributing to Nolan

Thank you for your interest in contributing to Nolan! This guide covers everything you need to get started.

---

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Environment Setup](#development-environment-setup)
- [Project Structure](#project-structure)
- [Pull Request Workflow](#pull-request-workflow)
- [Branch Naming](#branch-naming)
- [Commit Messages](#commit-messages)
- [Code Style](#code-style)
- [Testing Requirements](#testing-requirements)
- [Protobuf Schema Changes](#protobuf-schema-changes)
- [Security](#security)
- [Issue Templates](#issue-templates)
- [Getting Help](#getting-help)

---

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/1/code_of_conduct/). By participating, you agree to uphold a welcoming, harassment-free environment for everyone.

---

## Development Environment Setup

### Prerequisites

| Tool | Minimum Version | Purpose |
|------|----------------|---------|
| Docker | 24.0 | Container runtime |
| Docker Compose | 2.20 | Multi-container orchestration |
| Go | 1.22 | Control plane development |
| Rust | 1.78 | Worker development |
| Buf CLI | 1.30 | Protobuf linting and code generation |
| FFmpeg | 6.0 | Worker native builds (not needed for Docker) |
| Make | Any | Task runner |
| Git | 2.40 | Version control |

### First-Time Setup

```bash
# 1. Fork and clone
git clone https://github.com/<your-username>/nolan.git
cd nolan
git remote add upstream https://github.com/samaymehar/nolan.git

# 2. Generate development secrets
./scripts/gen-secrets.sh

# 3. Start infrastructure dependencies
docker compose up -d postgres valkey minio

# 4. Install Go dependencies
cd cmd/controlplane && go mod download && cd ../..

# 5. Install Rust dependencies
cd worker && cargo fetch && cd ..

# 6. Generate protobuf code
make proto

# 7. Verify everything works
make test
```

### IDE Recommendations

- **Go**: VS Code with the official Go extension, or GoLand.
- **Rust**: VS Code with rust-analyzer, or CLion with the Rust plugin.
- **Proto**: VS Code with the Buf extension for real-time linting.

---

## Project Structure

```
nolan/
├── cmd/controlplane/     # Go control plane entrypoint
├── internal/             # Go internal packages (not importable externally)
│   ├── api/              #   HTTP handlers and middleware
│   ├── auth/             #   JWT issuance and validation
│   ├── job/              #   Job orchestration logic
│   ├── storage/          #   MinIO client wrapper
│   └── valkey/           #   Valkey Streams client
├── worker/               # Rust worker crate
│   ├── src/
│   │   ├── main.rs       #     Entrypoint
│   │   ├── consumer.rs   #     Valkey consumer loop
│   │   ├── transcode.rs  #     FFmpeg pipeline
│   │   └── upload.rs     #     MinIO presigned upload
│   └── Cargo.toml
├── proto/nolan/v1/       # Protobuf definitions
├── scripts/              # Helper scripts (gen-secrets, migrations)
├── deploy/               # Kubernetes manifests (future)
├── docker/               # Dockerfiles
│   ├── controlplane.Dockerfile
│   └── worker.Dockerfile
├── docker-compose.yml
├── Makefile
├── buf.yaml
└── buf.gen.yaml
```

---

## Pull Request Workflow

1. **Create an issue first** (unless it's a trivial fix). Describe what you want to change and why.

2. **Create a branch** from `main`:
   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feat/your-feature-name
   ```

3. **Make your changes.** Keep commits focused and atomic.

4. **Run checks locally** before pushing:
   ```bash
   make lint    # Go + Rust + Proto linting
   make test    # Unit tests
   ```

5. **Push and open a PR:**
   ```bash
   git push origin feat/your-feature-name
   ```
   - Fill out the PR template.
   - Link the related issue.
   - Add a clear description of *what* changed and *why*.

6. **CI must pass.** The PR will not be reviewed until all CI checks are green.

7. **Address review feedback.** Push additional commits; do not force-push during review.

8. **Squash and merge.** Maintainers will squash-merge your PR into `main`.

### PR Checklist

- [ ] All CI checks pass (lint, test, build)
- [ ] New code has unit tests
- [ ] Significant changes have integration tests
- [ ] Proto changes pass `buf lint` and `buf breaking`
- [ ] No secrets or credentials in the diff
- [ ] CHANGELOG.md updated (under `## [Unreleased]`)
- [ ] Documentation updated if behavior changed

---

## Branch Naming

Use the following prefixes:

| Prefix | Use Case | Example |
|--------|----------|---------|
| `feat/` | New feature | `feat/dash-output-support` |
| `fix/` | Bug fix | `fix/presign-url-expiry` |
| `docs/` | Documentation only | `docs/architecture-valkey-schema` |
| `refactor/` | Code restructure, no behavior change | `refactor/split-job-handler` |
| `test/` | Test-only changes | `test/worker-retry-coverage` |
| `ci/` | CI/CD pipeline changes | `ci/add-arm64-build` |
| `chore/` | Dependency updates, tooling | `chore/bump-go-1.23` |

---

## Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>(<scope>): <short summary>

<optional body>

<optional footer(s)>
```

### Types

| Type | Description |
|------|-------------|
| `feat` | New feature |
| `fix` | Bug fix |
| `docs` | Documentation |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or fixing tests |
| `ci` | CI/CD changes |
| `chore` | Maintenance (deps, tooling) |
| `perf` | Performance improvement |
| `proto` | Protobuf schema change (always call this out explicitly) |

### Scopes

Use the component name: `controlplane`, `worker`, `proto`, `envoy`, `docker`, `ci`.

### Examples

```
feat(worker): add AV1 encoding profile support

Adds SVT-AV1 encoder configuration with CRF-based quality targeting.
Tested with 1080p60 source at CRF 30 — achieves 2.3× real-time on 4 vCPU.

Closes #42
```

```
fix(controlplane): prevent presigned URL reuse after job cancellation

Previously, presigned PUT URLs remained valid after a job was cancelled,
allowing orphaned uploads. Now the control plane checks job status before
minting presigned URLs.
```

```
proto(v1): add rendition_spec.max_framerate field

Non-breaking additive change. Workers that don't understand this field
will use the source framerate as before.

BREAKING: No
```

---

## Code Style

### Go (Control Plane)

- **Formatter**: `gofmt` (enforced by CI — zero-config).
- **Linter**: `go vet` + `golangci-lint` with the repo's `.golangci.yml` config.
- **Error handling**: Always wrap errors with `fmt.Errorf("context: %w", err)`. Never discard errors silently.
- **Naming**: Follow [Effective Go](https://go.dev/doc/effective_go) conventions. Exported names are PascalCase; unexported are camelCase.
- **Packages**: Keep packages small and focused. Use `internal/` to prevent external imports.
- **Context**: All functions that do I/O must accept `context.Context` as the first parameter.

```bash
# Run Go linters
golangci-lint run ./...
```

### Rust (Worker)

- **Formatter**: `rustfmt` with the repo's `rustfmt.toml` config.
- **Linter**: `clippy` at the `pedantic` lint level. All warnings are errors in CI.
- **Error handling**: Use `thiserror` for library errors and `anyhow` for application errors. No `unwrap()` in production code paths.
- **Naming**: Follow standard Rust conventions (snake_case for functions/variables, PascalCase for types).
- **Unsafe**: Prohibited unless absolutely necessary for FFmpeg FFI. Each `unsafe` block must have a `// SAFETY:` comment explaining the invariant.

```bash
# Run Rust linters
cd worker
cargo fmt --check
cargo clippy -- -D warnings
```

### Protobuf

- **Linter**: `buf lint` with the repo's `buf.yaml` config.
- **Style**: Follow the [Buf style guide](https://buf.build/docs/best-practices/style-guide/).
- **Field numbering**: Never reuse or reassign field numbers. Deleted fields must be `reserved`.
- **Naming**: `snake_case` for fields, `PascalCase` for messages and enums, `UPPER_SNAKE_CASE` for enum values.

```bash
# Run proto linter
buf lint proto/
```

---

## Testing Requirements

### Unit Tests (Required for All PRs)

- All new code must have unit tests.
- Tests must pass locally before pushing: `make test`.
- Aim for meaningful coverage, not 100% line coverage. Test behavior, not implementation.

```bash
# Go unit tests
go test ./... -race -count=1

# Rust unit tests
cd worker && cargo test
```

### Integration Tests (Required for Significant Changes)

"Significant" means changes to:
- Data flow between components (control plane ↔ Valkey ↔ worker ↔ MinIO)
- Authentication or authorization logic
- Protobuf serialization/deserialization
- Job lifecycle (submit → process → complete/fail)
- Failure recovery (retry, XCLAIM, DLQ)

Integration tests use Docker Compose to spin up real dependencies:

```bash
# Run integration tests (starts Valkey, MinIO, Postgres in Docker)
make test-integration
```

### What CI Checks

| Check | Tool | Blocking |
|-------|------|----------|
| Go lint | `golangci-lint` | Yes |
| Go tests | `go test -race` | Yes |
| Rust lint | `clippy -D warnings` | Yes |
| Rust format | `rustfmt --check` | Yes |
| Rust tests | `cargo test` | Yes |
| Proto lint | `buf lint` | Yes |
| Proto breaking | `buf breaking --against .git#branch=main` | Yes |
| Secret scan | `gitleaks` | Yes |
| Docker build | `docker compose build` | Yes |
| Integration tests | `make test-integration` | Yes (on `main` and release branches) |

---

## Protobuf Schema Changes

Proto changes are high-impact because they affect the contract between the control plane and workers.

### Rules

1. **Additive changes only** within a major version. Add new fields; never remove or renumber existing ones.
2. **Deleted fields must be `reserved`** to prevent accidental reuse.
3. **All changes must pass both linters:**
   ```bash
   buf lint proto/
   buf breaking proto/ --against .git#branch=main
   ```
4. **Run code generation** after any proto change:
   ```bash
   make proto
   ```
5. **Update both Go and Rust code** to handle new fields. The PR must include generated code updates.
6. **Document the change** in CHANGELOG.md under a `### Proto Changes` subsection.
7. **Breaking changes** (field removal, type change, renumbering) require:
   - A new proto package version (`nolan.v2`)
   - A migration guide
   - Approval from two maintainers

---

## Security

### No Secrets in Commits

- The `.env` file is git-ignored. Never commit it.
- CI runs `gitleaks` on every PR. If it detects a potential secret, the PR will be blocked.
- If you accidentally commit a secret:
  1. **Rotate the secret immediately.**
  2. Use `git filter-branch` or `BFG Repo-Cleaner` to remove it from history.
  3. Notify maintainers.

### Vulnerability Reports

Do **not** open a public issue for security vulnerabilities. See [SECURITY.md](SECURITY.md) for the responsible disclosure process.

---

## Issue Templates

When opening an issue, please use one of the provided templates:

- **Bug Report**: Describe what happened, what you expected, and steps to reproduce.
- **Feature Request**: Describe the use case, proposed solution, and alternatives considered.
- **Proto Change Proposal**: Describe the schema change, backward compatibility impact, and migration plan.

If no template fits, open a blank issue with a clear title and description.

---

## Getting Help

- **Discussions**: Use [GitHub Discussions](https://github.com/samaymehar/nolan/discussions) for questions, ideas, and design conversations.
- **Issues**: Use [GitHub Issues](https://github.com/samaymehar/nolan/issues) for bugs and feature requests.
- **Chat**: Join us on the `#nolan` channel (link in repo description).

We're happy to help new contributors find a good first issue. Look for issues labeled `good first issue` or `help wanted`.

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

> **Proto contract changes** are always listed in their own subsection because they affect the control-plane ↔ worker interface and may require coordinated deployments.

---

## [Unreleased]

### Added

- **Go control plane** with REST API for user auth, job management, and asset serving.
- **Rust worker** with FFmpeg-based transcoding and Valkey Streams consumer.
- **Valkey Streams** job queue with consumer groups, XCLAIM recovery, and DLQ.
- **MinIO integration** with presigned URL upload/download for source and output media.
- **Envoy edge proxy** with TLS termination, JWT validation, and segment caching.
- **PostgreSQL** as the system of record for users, assets, and jobs.
- **Docker Compose** one-command deployment for the full stack.
- **HLS output** with adaptive bitrate renditions (720p, 480p, 360p).
- **JWT authentication** with access and refresh token support.
- **Presigned URL security model** — workers never hold MinIO credentials.
- **Idempotent workers** — at-least-once delivery is safe due to deterministic output and key overwrite.
- **Worker heartbeat and XCLAIM recovery** — stale workers detected within 30 seconds.
- **Dead-letter queue** for permanently failed jobs after configurable retry count.
- **Prometheus metrics endpoint** for monitoring.
- **Health and readiness probes** (`/healthz`, `/readyz`).
- **`gen-secrets.sh`** and **`gen-secrets.ps1`** for development secret generation.
- **Multi-arch Docker images** (amd64 / arm64).
- **Non-root containers** with read-only root filesystems.
- **Network isolation** — only Envoy is exposed to the host; all other services on internal network.
- **VP9, AV1 (SVT-AV1), Opus, AAC** codecs in the default permissive build.
- **Optional GPL codec support** (x264, x265) via `GPL_CODECS=1` build flag.

### Proto Changes

- Initial `nolan.v1` protobuf contract:
  - `TranscodeJob` — job message with source key, output prefix, and encoding profile.
  - `TranscodeProfile` — codec, container, and rendition specifications.
  - `RenditionSpec` — width, height, bitrate, and max framerate per variant.
  - `VideoCodec` enum — VP9, AV1, H.264 (GPL), H.265 (GPL).
  - `AudioCodec` enum — Opus, AAC.
  - `ContainerFormat` enum — HLS, DASH.

### Documentation

- `README.md` with architecture diagram, quickstart, API reference, and honest caveats.
- `ARCHITECTURE.md` with component boundaries, data flow, key schema, and security model.
- `CONTRIBUTING.md` with dev setup, PR workflow, code style, and testing requirements.
- `SECURITY.md` with vulnerability disclosure process and scope.
- `LICENSE` — Apache License, Version 2.0.

---

## Versioning Policy

### Semantic Versioning

Nolan follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (1.0.0 → 2.0.0): Breaking changes to the REST API or protobuf contract.
- **MINOR** (0.1.0 → 0.2.0): New features, additive proto changes, non-breaking API additions.
- **PATCH** (0.1.0 → 0.1.1): Bug fixes, security patches, documentation updates.

### Pre-1.0 Stability

While Nolan is below 1.0.0, minor versions may include breaking changes. We will clearly document these in the changelog and provide migration guides where possible.

### Proto Contract Versioning

Protobuf contract changes are versioned independently within the proto package namespace:

- **Additive changes** (new fields, new enum values) are backward-compatible and do not require a major version bump.
- **Breaking changes** (field removal, type changes, renumbering) require a new proto package version (`nolan.v1` → `nolan.v2`) and a coordinated migration.

Proto changes are always highlighted in their own `### Proto Changes` subsection in the changelog to make them easy to find during upgrade planning.

---

<!-- Links -->
[Unreleased]: https://github.com/samaymehar/nolan/compare/HEAD

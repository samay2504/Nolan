# Architecture

This document describes the internal architecture of Nolan. It is intended for contributors, operators, and anyone evaluating the system for production use.

> **Scope.** This covers the v0.1 architecture. Components marked *(planned)* are not yet implemented.

---

## Table of Contents

- [System Overview](#system-overview)
- [Component Diagram](#component-diagram)
- [Component Responsibilities](#component-responsibilities)
- [Data Flow](#data-flow)
- [Valkey Key Schema](#valkey-key-schema)
- [MinIO Storage Layout](#minio-storage-layout)
- [Protobuf Contract](#protobuf-contract)
- [Failure Handling](#failure-handling)
- [Security Model](#security-model)
- [Hard Boundaries](#hard-boundaries)

---

## System Overview

Nolan is a distributed video transcoding pipeline with five core components:

1. **Envoy Edge Proxy** — ingress, TLS, JWT validation, segment caching
2. **Go Control Plane** — API server, job orchestration, metadata management
3. **Valkey** — ephemeral job queue via Streams + consumer groups
4. **Rust Workers** — FFmpeg-based transcoding engines
5. **MinIO** — S3-compatible object storage for source and output media

PostgreSQL provides durable state for the control plane but is not on the hot path for media delivery.

---

## Component Diagram

```
                                  Internet
                                     │
                              ┌──────▼──────┐
                              │   Envoy     │
                              │  ─────────  │
                              │  TLS term.  │
                              │  JWT verify │
                              │  Seg. cache │
                              └──┬──────┬───┘
                                 │      │
                    API routes   │      │  Media routes
                                 │      │  (presigned redirect)
                    ┌────────────▼─┐  ┌─▼────────────────┐
                    │  Go Control  │  │     MinIO         │
                    │    Plane     │  │  ────────────     │
                    │  ──────────  │  │  Source bucket    │
                    │  REST API    │  │  Output bucket    │
                    │  Job submit  │  │  Presigned GET    │
                    │  Presigned   │  │  Presigned PUT    │
                    │  URL mint    │  │                   │
                    └──┬───────┬──┘  └──▲────────────────┘
                       │       │        │
              ┌────────▼──┐    │        │
              │ PostgreSQL │    │        │
              │ ────────── │    │        │
              │ Users      │    │        │
              │ Assets     │    │        │
              │ Jobs       │    │        │
              └────────────┘    │        │
                       ┌────────▼──┐     │
                       │  Valkey   │     │
                       │  ──────── │     │
                       │  Streams  │     │
                       │  Consumer │     │
                       │  groups   │     │
                       └────┬──────┘     │
                            │            │
                    ┌───────▼──────────┐  │
                    │  Rust Worker(s)  │──┘
                    │  ──────────────  │
                    │  XREADGROUP      │
                    │  FFmpeg decode   │
                    │  FFmpeg encode   │
                    │  Presigned PUT   │
                    │  XACK on done    │
                    └──────────────────┘
```

---

## Component Responsibilities

### Envoy Edge Proxy

| Does | Does NOT |
|------|----------|
| TLS termination (HTTPS → HTTP) | Hold any application state |
| JWT signature verification (HMAC-SHA256) | Issue or refresh tokens |
| Route API calls to the control plane | Perform video transcoding |
| Serve cached media segments | Authenticate users (only validates existing tokens) |
| Rate limiting (connection + request) | Access MinIO directly for uploads |
| gRPC-Web bridging *(planned)* | |

### Go Control Plane

| Does | Does NOT |
|------|----------|
| User registration and authentication (JWT) | Transcode video |
| Job creation, status tracking, cancellation | Store media blobs (delegates to MinIO) |
| Mint presigned URLs for MinIO uploads and downloads | Manage Valkey consumer groups (workers self-register) |
| Publish jobs to Valkey Streams (`XADD`) | Read from Valkey Streams |
| Persist job state to PostgreSQL | Cache media segments |
| Serve asset manifests (HLS/DASH) | Perform TLS termination |
| Expose Prometheus metrics | |

### Valkey (Streams)

| Does | Does NOT |
|------|----------|
| Buffer job messages in an ordered, persistent stream | Provide exactly-once delivery (at-least-once only) |
| Support consumer groups for competing-consumer pattern | Store job metadata long-term (ephemeral) |
| Allow message reclaim via `XCLAIM` for stuck consumers | Authenticate or authorize clients |
| Track per-consumer acknowledgement (`XACK`) | Run any application logic |
| Provide `XPENDING` for monitoring unacknowledged messages | Replace PostgreSQL as system of record |

> **Naming note.** The in-memory data store is [Valkey](https://valkey.io/). The Rust worker uses the `redis` crate for protocol compatibility — this is a crate naming convention, not a runtime dependency on Redis.

### Rust Workers

| Does | Does NOT |
|------|----------|
| Consume jobs from Valkey via `XREADGROUP` | Accept HTTP requests |
| Download source media via presigned GET URL | Mint presigned URLs (receives them in the job message) |
| Transcode using FFmpeg (libavcodec / libavformat) | Write to PostgreSQL |
| Zero-copy buffer passing between FFmpeg stages (in-process) | Manage user authentication |
| Upload output segments via presigned PUT URL | Make zero-copy network transfers (I/O always copies) |
| Report progress to Valkey (hash key) | |
| `XACK` on completion or move to DLQ on permanent failure | |

### MinIO

| Does | Does NOT |
|------|----------|
| Store source uploads and transcoded output | Run any application logic |
| Enforce access via presigned URL signatures | Perform transcoding |
| Support S3-compatible API for interoperability | Provide a CDN (use Envoy cache or external CDN) |
| Bucket lifecycle policies for cleanup *(planned)* | Index or query metadata (use PostgreSQL) |

### PostgreSQL

| Does | Does NOT |
|------|----------|
| Persist users, assets, jobs, and their relationships | Queue or dispatch work (Valkey does this) |
| Provide ACID transactions for state changes | Store media blobs |
| Support migrations via `golang-migrate` | Get queried on the media delivery hot path |

---

## Data Flow

### Upload and Transcode

```
1. Client ──POST /api/v1/uploads/presign──▶ Envoy ──▶ Control Plane
   ◀── { upload_url: "https://minio/nolan-source/..." } ──◀

2. Client ──PUT (presigned)──▶ MinIO
   Source file lands in  nolan-source/<user_id>/<upload_id>/<filename>

3. Client ──POST /api/v1/jobs──▶ Envoy ──▶ Control Plane
   Control plane:
     a. Validates the upload exists (HEAD on MinIO)
     b. Inserts job row into PostgreSQL (status: PENDING)
     c. XADDs job message to Valkey stream  jobs:transcode

4. Rust Worker:
     a. XREADGROUP picks up the job
     b. Downloads source via presigned GET
     c. Runs FFmpeg transcode pipeline (in-process, zero-copy between stages)
     d. Uploads each segment via presigned PUT to MinIO
     e. Uploads manifest (m3u8 / mpd) to MinIO
     f. Publishes progress updates to Valkey hash
     g. XACKs the stream message
     h. Notifies control plane of completion (Valkey pub/sub or polling)

5. Control Plane:
     a. Updates PostgreSQL job row (status: COMPLETE, output_manifest_key)
     b. Asset is now playable

6. Client ──GET /api/v1/assets/:id/manifest──▶ Envoy ──▶ Control Plane
   ◀── HLS manifest with presigned segment URLs ──◀

7. Client (player) ──GET segment──▶ Envoy (cache check)
     Cache HIT  → serve from Envoy memory
     Cache MISS → presigned redirect to MinIO → cache and serve
```

### Playback Hot Path

```
Client ──▶ Envoy ──▶ (cache?) ──▶ MinIO
                        │
                  If cached: 0 backend hops
                  If miss:   1 redirect to MinIO
```

The control plane is **not on the playback hot path** after the manifest is served. Segment requests go directly through Envoy to MinIO.

---

## Valkey Key Schema

| Key Pattern | Type | TTL | Purpose |
|-------------|------|-----|---------|
| `jobs:transcode` | Stream | None (trimmed by MAXLEN) | Primary job queue. Messages contain serialized `TranscodeJob` proto. |
| `jobs:dlq` | Stream | None | Dead-letter queue for permanently failed jobs. |
| `job:progress:<job_id>` | Hash | 24 h | Worker writes: `percent`, `stage`, `fps`, `eta_seconds`. Control plane reads for status API. |
| `job:lock:<job_id>` | String (SET NX) | 5 min | Distributed lock to prevent double-processing during XCLAIM races. |
| `worker:heartbeat:<worker_id>` | String | 30 s (renewed) | Worker liveness signal. Control plane monitors for stale workers. |
| `metrics:jobs:completed` | String (INCR) | None | Counter for completed jobs. Scraped by Prometheus exporter. |
| `metrics:jobs:failed` | String (INCR) | None | Counter for failed jobs. |

### Stream Message Fields (`jobs:transcode`)

| Field | Type | Description |
|-------|------|-------------|
| `job_id` | string (UUID) | Unique job identifier, matches PostgreSQL PK |
| `user_id` | string (UUID) | Owning user |
| `source_key` | string | MinIO object key for source media |
| `output_prefix` | string | MinIO key prefix for output segments |
| `profile` | string | Encoding profile name (e.g., `hls-vp9-720p`) |
| `presign_get` | string | Presigned GET URL for source download |
| `presign_put_prefix` | string | Presigned PUT URL template for output upload |
| `proto_payload` | bytes | Full `TranscodeJob` protobuf for forward compatibility |
| `created_at` | string (RFC 3339) | Job creation timestamp |

---

## MinIO Storage Layout

```
nolan-source/
  └── <user_id>/
      └── <upload_id>/
          └── <original_filename>          # Raw uploaded file

nolan-output/
  └── <user_id>/
      └── <job_id>/
          ├── manifest.m3u8               # HLS master playlist
          ├── 720p/
          │   ├── playlist.m3u8           # Variant playlist
          │   ├── segment-000.ts          # Media segments
          │   ├── segment-001.ts
          │   └── ...
          ├── 480p/
          │   ├── playlist.m3u8
          │   └── ...
          └── 360p/
              ├── playlist.m3u8
              └── ...
```

**Bucket policies:**
- `nolan-source`: No public access. Presigned PUT for upload, presigned GET for worker download.
- `nolan-output`: No public access. Presigned GET minted by control plane for playback.
- Lifecycle rule *(planned)*: Delete source files 7 days after successful transcode.

---

## Protobuf Contract

Proto files live in `proto/nolan/v1/`. The contract is versioned and linted with `buf`.

### Key Message Types

```protobuf
// proto/nolan/v1/job.proto

message TranscodeJob {
  string job_id = 1;
  string user_id = 2;
  string source_key = 3;
  string output_prefix = 4;
  TranscodeProfile profile = 5;
  google.protobuf.Timestamp created_at = 6;
}

message TranscodeProfile {
  string name = 1;                    // e.g., "hls-vp9-720p"
  VideoCodec video_codec = 2;
  AudioCodec audio_codec = 3;
  ContainerFormat container = 4;
  repeated RenditionSpec renditions = 5;
}

message RenditionSpec {
  uint32 width = 1;
  uint32 height = 2;
  uint32 bitrate_kbps = 3;
  uint32 max_framerate = 4;
}

enum VideoCodec {
  VIDEO_CODEC_UNSPECIFIED = 0;
  VIDEO_CODEC_VP9 = 1;
  VIDEO_CODEC_AV1 = 2;
  VIDEO_CODEC_H264 = 3;    // Requires GPL opt-in
  VIDEO_CODEC_H265 = 4;    // Requires GPL opt-in
}

enum AudioCodec {
  AUDIO_CODEC_UNSPECIFIED = 0;
  AUDIO_CODEC_OPUS = 1;
  AUDIO_CODEC_AAC = 2;
}

enum ContainerFormat {
  CONTAINER_FORMAT_UNSPECIFIED = 0;
  CONTAINER_FORMAT_HLS = 1;     // fMP4 or MPEG-TS segments
  CONTAINER_FORMAT_DASH = 2;    // fMP4 segments
}
```

### Contract Rules

- **Additive only.** New fields are added; existing fields are never removed or renumbered.
- **Proto breaking changes** require a major version bump (`v1` → `v2`) and a migration path.
- All proto changes must pass `buf lint` and `buf breaking --against .git#branch=main`.
- The `proto_payload` field in Valkey stream messages carries the full serialized proto. Workers deserialize this and ignore the flat fields (which exist for observability tooling that cannot parse protobuf).

---

## Failure Handling

### Idempotency

Workers are fully idempotent. Re-processing a job:
1. Downloads the same source file.
2. Produces deterministic output (FFmpeg with fixed seed / no timestamp-based randomness).
3. Overwrites the same MinIO keys.
4. Results in the same PostgreSQL state.

This means at-least-once delivery from Valkey is safe — duplicate processing wastes compute but does not corrupt state.

### Worker Crash Recovery (XCLAIM)

```
1. Worker A picks up job J via XREADGROUP
2. Worker A crashes mid-transcode
3. Worker A's heartbeat key expires (30s TTL)
4. Control plane detects stale heartbeat
5. Control plane runs XCLAIM to reassign job J's pending message
   to a healthy worker's consumer name
6. Healthy worker picks up job J on next XREADGROUP
7. Idempotent processing ensures correct output
```

**Safeguards:**
- `XCLAIM` requires a minimum idle time (60 s) to avoid reclaiming messages from slow-but-alive workers.
- The `job:lock:<job_id>` key prevents two workers from racing on the same job after an XCLAIM. The lock is acquired with `SET NX EX 300`. If the lock is held, the worker skips the job and it will be reclaimed again later.

### Dead-Letter Queue (DLQ)

Jobs that fail more than **3 times** (configurable via `NOLAN_WORKER_MAX_RETRIES`) are moved to the `jobs:dlq` stream:

1. Worker increments a retry counter in the job progress hash.
2. On the Nth failure, worker `XADD`s the message to `jobs:dlq` with failure metadata.
3. Worker `XACK`s the original message in `jobs:transcode` to remove it from the pending list.
4. Control plane updates the PostgreSQL job row to `status: FAILED` with the error message.
5. Admin can inspect the DLQ and manually retry or discard failed jobs.

### Failure Modes Summary

| Failure | Detection | Recovery | Data Loss Risk |
|---------|-----------|----------|----------------|
| Worker crash | Heartbeat expiry (30 s) | XCLAIM + idempotent reprocess | None (at-least-once) |
| Worker OOM | Container restart + heartbeat | Same as crash | None |
| MinIO upload failure | HTTP error from presigned PUT | Worker retries with exponential backoff (3 attempts) | None if retries succeed; job fails to DLQ otherwise |
| Valkey crash | Control plane health check | Valkey restarts (AOF persistence); pending jobs are re-read from stream | Possible loss of in-flight messages if AOF is async. Use `appendfsync everysec` for durability. |
| PostgreSQL crash | Control plane connection error | Postgres restarts; WAL ensures durability | None (WAL + fsync) |
| Control plane crash | Envoy health check (503) | Container restart; stateless service recovers immediately | None (state is in Postgres) |
| Envoy crash | Docker restart policy | Container restart; clients reconnect | Cache is lost; refilled on next requests |

---

## Security Model

### Network Isolation

```
┌─────────────────── Docker Network: nolan-public ───────────────────┐
│   Envoy (port 8443 exposed to host)                                │
└──────────────────────────┬─────────────────────────────────────────┘
                           │
┌──────────────────────────▼──── Docker Network: nolan-internal ─────┐
│   Control Plane    Valkey    MinIO    PostgreSQL    Workers         │
│   (no ports exposed to host)                                       │
└────────────────────────────────────────────────────────────────────┘
```

- **Only Envoy** is exposed to the host network.
- All internal services communicate over the `nolan-internal` bridge network.
- Valkey, PostgreSQL, and MinIO are not reachable from outside the Docker network.

### Authentication and Authorization

| Layer | Mechanism |
|-------|-----------|
| Client → Envoy | TLS 1.3 (self-signed in dev, ACME/Let's Encrypt in production) |
| Envoy → Control Plane | JWT signature verification at Envoy; control plane trusts `X-User-Id` header |
| Control Plane → MinIO | Static access key / secret key (rotated via `gen-secrets`) |
| Control Plane → Valkey | Password authentication (`requirepass`) |
| Control Plane → PostgreSQL | Username/password over TLS *(planned: mTLS)* |
| Worker → Valkey | Same password as control plane |
| Worker → MinIO | Presigned URLs only — workers do not hold MinIO credentials |
| Client → MinIO (segments) | Presigned GET URLs with short TTL (15 min default) |

### Presigned URL Security

- **Short-lived.** Default TTL is 15 minutes, configurable via `NOLAN_MINIO_PRESIGN_TTL`.
- **Scoped.** Each URL is scoped to a specific object key and HTTP method (GET or PUT).
- **Non-transferable.** URLs include a signature derived from the MinIO secret key. Tampering with the path or query parameters invalidates the signature.
- **No credential leakage.** Workers never receive MinIO access keys. They receive presigned URLs in the job message and use them for uploads.

### Container Hardening

- All containers run as **non-root** users.
- Worker containers have **no network access** except to MinIO and Valkey (via Docker network policy).
- Worker containers mount **no host volumes** — media is transferred via presigned URLs over HTTP.
- Read-only root filesystem on all containers except MinIO (which needs writable data volume).
- Resource limits (CPU, memory) set in `docker-compose.yml` to prevent noisy-neighbor effects.

### Secrets Management

- Secrets are generated by `scripts/gen-secrets.sh` and stored in `.env` (git-ignored).
- The `.env` file is the single source of truth for all secrets in development.
- For production, use Docker Secrets, Vault, or your platform's native secrets manager.
- `gitleaks` runs in CI to prevent accidental secret commits.

---

## Hard Boundaries

These are architectural decisions that are **intentionally fixed** and will not change without a major version bump.

| Boundary | Rationale |
|----------|-----------|
| **Valkey is ephemeral, PostgreSQL is durable.** Job state lives in PostgreSQL. Valkey is a transport. If Valkey data is lost, pending jobs can be re-enqueued from PostgreSQL. | Separation of concerns: queues should be fast and disposable; state should be durable and queryable. |
| **Workers never talk to PostgreSQL.** Workers read from Valkey and write to MinIO. All state updates flow through the control plane. | Workers are untrusted compute. Limiting their blast radius simplifies security and makes scaling trivial. |
| **Workers never hold MinIO credentials.** They receive presigned URLs. | Minimizes credential surface. A compromised worker cannot enumerate or delete objects. |
| **Envoy is the only public-facing component.** | Single ingress point simplifies TLS, rate limiting, and audit logging. |
| **Protobuf is the wire format between control plane and workers.** | Schema evolution, backward compatibility, and efficient serialization. JSON is used only for the REST API layer. |
| **At-least-once delivery, never exactly-once.** | Exactly-once is impossible in distributed systems without two-phase commit. Idempotent workers make at-least-once safe and simple. |
| **Control plane is stateless.** | Any instance can serve any request. Horizontal scaling is trivial. All state lives in PostgreSQL and Valkey. |

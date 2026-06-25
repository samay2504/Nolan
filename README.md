<p align="center">
  <strong>Nolan</strong><br>
  Self-hosted, open-source distributed video transcoding pipeline.
</p>

<p align="center">
  <a href="LICENSE">Apache-2.0</a> ·
  <a href="ARCHITECTURE.md">Architecture</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="SECURITY.md">Security</a> ·
  <a href="CHANGELOG.md">Changelog</a>
</p>

---

## Features

- **Distributed video transcoding** with adaptive bitrate output (HLS / DASH)
- **Modern fMP4 Web Playback** — uses `fMP4` with VP9/Opus codecs for native modern browser support via `hls.js`.
- **Go control plane** — authentication, metadata, job orchestration via Valkey Streams
- **Rust workers** — high-performance FFmpeg transcoding with parallel processing limits
- **MinIO object storage** — S3-compatible, secured with presigned URLs
- **Envoy edge proxy** — TLS termination, JWT validation, and HLS segment caching
- **PostgreSQL** — system of record for users, assets, and job state
- **Docker Compose** — one command to run the full stack

## Architecture

```text
                               ┌──────────────────────┐
                               │       Client         │
                               └──────────┬───────────┘
                                          │ HTTP
                               ┌──────────▼───────────┐
                               │   Envoy Edge Proxy   │
                               │  CORS · Routes · Cache│
                               └──────────┬───────────┘
                                          │
                     ┌────────────────────┬┘
                     │                    │
          ┌──────────▼──────────┐  ┌─────▼──────────────┐
          │  Go Control Plane   │  │   MinIO: Objects   │
          │  - Auth & metadata  │  │   - Source uploads │
          │  - Presigned URLs   │  │   - fMP4 / HLS     │
          │  - Valkey XADD      │  │   - Presigned GET  │
          └──────────┬──────────┘  └─────▲──────────────┘
                     │                    │
          ┌──────────▼──────────┐         │
          │  Valkey: Streams    │         │
          │  - Job queue        │         │
          │  - Consumer groups  │         │
          └──────────┬──────────┘         │
                     │                    │
          ┌──────────▼──────────┐         │
          │   Rust Workers      │─────────┘
          │  - FFmpeg transcode │
          │  - Upload segments  │
          │  - Progress report  │
          └─────────────────────┘
```

**Data flows down.** Transcoded fMP4 segments flow right to MinIO, then Envoy serves them to clients. PostgreSQL backs the control plane for durable state; Valkey is the ephemeral job bus.

## Quick Start & Developer Guide

### 1. Clone & Setup Secrets
First, clone the repository and generate your local development secrets.

```bash
git clone https://github.com/samaymehar/nolan.git
cd nolan

# Generate .env file secrets
./scripts/gen-secrets.sh      # Linux / macOS
# .\scripts\gen-secrets.ps1   # Windows (PowerShell)
```

### 2. Start the Product Environment
Docker Compose brings up the entire distributed system (Envoy, Control Plane, Valkey, MinIO, Postgres, and the Rust Workers).

```bash
docker compose up -d
```
On your first run, Docker will compile the Go and Rust binaries inside multi-stage builds.

### 3. Verify System Health
Check that the Envoy proxy and Control Plane are routing properly:
```bash
curl http://localhost:8443/healthz
```

### 4. Transcode a Video!
We have included automated testing scripts to simulate the frontend API flow. To test transcoding a video, use the End-to-End PowerShell script:

```bash
# Windows (PowerShell)
.\scripts\e2e-test.ps1 "D:\path\to\your\video.mp4"
```
*(If you do not provide a path, it will default to a fallback video path)*

This script will:
1. Request a pre-signed URL from the Control Plane.
2. Upload the raw `.mp4` into MinIO.
3. Queue a job into Valkey.
4. Wait for the Rust Worker to pick up the job and transcode it to 480p and 720p Fragmented MP4 (fMP4) HLS streams.

Once it completes, the script will output the `master.m3u8` URLs that are accessible via `http://localhost:8443`.

### 5. Stress Testing the Architecture
To verify load balancing and the `WORKER_CONCURRENCY` limits, you can run the stress test script which uploads 4 videos simultaneously:
```bash
# Windows (PowerShell)
.\scripts\stress-test.ps1
```
Use `docker stats` during the test to watch the multiple `nolan-worker` containers perfectly balance the workload!

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/healthz` | Liveness probe — returns `200 OK` |
| `POST` | `/api/v1/uploads/presign` | Get a presigned PUT URL for source upload to MinIO |
| `POST` | `/api/v1/jobs` | Submit a transcoding job to the Valkey stream |
| `GET` | `/media/:id/hls/:resolution/master.m3u8` | Public read access to the transcoded HLS stream (served by Envoy + MinIO) |

*(Authentication is simplified in the MVP. Refer to `control-plane/main.go` to see the route configurations).*

## Configuration

All configuration is via environment variables. Defaults are development-friendly and located in the `.env` file generated by the `gen-secrets` script.

| Variable | Default | Description |
|----------|---------|-------------|
| `NOLAN_LISTEN_ADDR` | `:8080` | Control plane HTTP listen address |
| `NOLAN_POSTGRES_DSN` | `postgres://nolan:nolan@postgres:5432/nolan` | PostgreSQL connection string |
| `NOLAN_VALKEY_ADDR` | `valkey:6379` | Valkey server address |
| `NOLAN_VALKEY_STREAM` | `pipeline:jobs:transcode` | Valkey stream key for job dispatch |
| `NOLAN_WORKER_CONCURRENCY` | `2` | Number of parallel transcode jobs per worker container |

## Codec Licensing & Browser Playback

Nolan heavily uses modern, permissive codecs by default: **VP9** and **Opus**.
To ensure that these codecs play perfectly in modern web browsers (Chrome, Firefox, Safari), Nolan has been specifically engineered to output **Fragmented MP4 (fMP4)** segments (`.m4s` and `init.mp4`) instead of legacy `.ts` segments. 

To play your transcoded video links in Chrome, simply pass the `.m3u8` link into a player library like [hls.js](https://github.com/video-dev/hls.js/) or use a native HTML5 video element on Safari.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).

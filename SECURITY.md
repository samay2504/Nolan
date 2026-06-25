# Security Policy

The Nolan project takes security seriously. This document describes how to report vulnerabilities, what is in scope, and what to expect from the response process.

---

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x (current development) | ✅ Active development — security fixes applied promptly |
| < 0.1.0 | ❌ Pre-release — no security support |

Once Nolan reaches 1.0, this table will be updated to reflect a formal support window (e.g., current major and previous major).

---

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please report vulnerabilities via email:

📧 **security@nolan-project.dev**

### What to Include

To help us triage and fix the issue quickly, please include:

1. **Description**: A clear description of the vulnerability and its potential impact.
2. **Affected component**: Which part of Nolan is affected (control plane, worker, Envoy config, proto, Docker setup).
3. **Reproduction steps**: Step-by-step instructions to reproduce the vulnerability.
4. **Environment**: OS, Docker version, Nolan version/commit, and any relevant configuration.
5. **Proof of concept**: If you have one, include it. Code, logs, or screenshots are all helpful.
6. **Suggested fix**: If you have ideas on how to fix it, we welcome them.

### PGP Encryption (Optional)

If you want to encrypt your report, you can use the project's PGP key (published in the repo at `SECURITY_PGP_KEY.asc` once available). Until then, plaintext email is acceptable.

---

## Response Timeline

| Phase | Target Time |
|-------|-------------|
| **Acknowledgement** | Within 48 hours of report |
| **Initial triage** | Within 5 business days |
| **Fix development** | Within 14 business days for critical/high severity |
| **Coordinated disclosure** | Within 90 days of report, or sooner if fix is released |

These are targets, not guarantees. Complex issues may take longer. We will keep you updated on progress.

### Severity Classification

| Severity | Description | Example |
|----------|-------------|---------|
| **Critical** | Remote code execution, full system compromise | Arbitrary command injection in worker FFmpeg args |
| **High** | Unauthorized data access, privilege escalation | JWT bypass allowing access to other users' assets |
| **Medium** | Limited data exposure, denial of service | Presigned URL with excessive TTL, Valkey stream flood |
| **Low** | Information disclosure with minimal impact | Version string leakage, verbose error messages |

---

## Scope

### In Scope

The following are considered valid security concerns:

- **Authentication and authorization bypass** in the control plane (JWT, API endpoints)
- **Presigned URL vulnerabilities** (excessive scope, missing expiry, replay attacks)
- **Container escape** or privilege escalation in worker or control plane containers
- **Injection attacks** — command injection in FFmpeg arguments, SQL injection in control plane
- **Secrets exposure** — credentials leaked in logs, error messages, or API responses
- **Denial of service** — resource exhaustion via malicious input files, stream flooding
- **Dependency vulnerabilities** — known CVEs in direct dependencies (Go modules, Rust crates, Docker base images)
- **Docker Compose configuration** — insecure defaults, exposed ports, missing network isolation
- **Envoy misconfiguration** — TLS downgrade, missing rate limits, header injection
- **Prototype pollution or deserialization attacks** in protobuf handling

### Out of Scope

The following are **not** considered security vulnerabilities in Nolan:

- **Self-hosted deployment hardening.** Nolan provides secure defaults, but operators are responsible for their infrastructure (firewall rules, OS patches, disk encryption).
- **MinIO or PostgreSQL vulnerabilities.** Report these to the respective upstream projects. However, if our *configuration* of these services is insecure, that is in scope.
- **Social engineering** of project maintainers or contributors.
- **Attacks requiring physical access** to the host machine.
- **Denial of service via resource limits.** If you can crash a worker by submitting a 100 GB file and the operator has not set resource limits, that is an operational concern, not a Nolan vulnerability.
- **Vulnerabilities in optional GPL-licensed codecs** (x264, x265) that are not built by default. Report these upstream.

---

## Disclosure Policy

We follow a **coordinated disclosure** process:

1. **Reporter submits** a vulnerability via email.
2. **We acknowledge** the report and assign a tracking ID.
3. **We develop and test a fix** in a private branch.
4. **We coordinate with the reporter** on disclosure timing.
5. **We release the fix** with a security advisory on GitHub.
6. **We credit the reporter** (unless they prefer to remain anonymous).

We aim to release fixes before public disclosure. If we cannot fix the issue within 90 days, we will work with the reporter to agree on a disclosure timeline.

---

## Security Best Practices for Operators

While not part of the vulnerability reporting process, here are key security practices for anyone deploying Nolan:

- **Rotate secrets regularly.** Re-run `gen-secrets.sh` and restart services.
- **Use TLS in production.** The default Docker Compose uses self-signed certificates. Replace with ACME/Let's Encrypt for production.
- **Set resource limits.** Configure CPU and memory limits in `docker-compose.yml` or your orchestrator.
- **Enable Valkey authentication.** Set `requirepass` in Valkey configuration (done by default in our Compose file).
- **Monitor the DLQ.** Failed jobs in the dead-letter queue may indicate malicious input.
- **Keep images updated.** Pull the latest Nolan images regularly to get security patches.
- **Review presigned URL TTLs.** The default 15-minute TTL is suitable for most use cases. Shorter is better for high-security environments.

---

## Recognition

We gratefully acknowledge security researchers who help us improve Nolan. With the reporter's permission, we will list them here:

*No reports yet — be the first!*

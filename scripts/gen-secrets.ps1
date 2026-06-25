$ErrorActionPreference = "Stop"

function Get-RandomHex {
    param([int]$Bytes)
    $rnd = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $buffer = New-Object byte[] $Bytes
    $rnd.GetBytes($buffer)
    return [System.BitConverter]::ToString($buffer).Replace("-", "").ToLower()
}

$VALKEY_DEFAULT_PASSWORD = Get-RandomHex -Bytes 16
$CONTROLPLANE_VALKEY_PASSWORD = Get-RandomHex -Bytes 16
$WORKER_VALKEY_PASSWORD = Get-RandomHex -Bytes 16
$MINIO_ROOT_PASSWORD = Get-RandomHex -Bytes 16
$POSTGRES_PASSWORD = Get-RandomHex -Bytes 16

$envContent = @"
# Valkey
VALKEY_DEFAULT_PASSWORD=$VALKEY_DEFAULT_PASSWORD
CONTROLPLANE_VALKEY_PASSWORD=$CONTROLPLANE_VALKEY_PASSWORD
WORKER_VALKEY_PASSWORD=$WORKER_VALKEY_PASSWORD

# MinIO
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=$MINIO_ROOT_PASSWORD
MINIO_ENDPOINT=minio:9000

# PostgreSQL
POSTGRES_DB=nolan
POSTGRES_USER=nolan
POSTGRES_PASSWORD=$POSTGRES_PASSWORD
DATABASE_URL=postgres://nolan:$POSTGRES_PASSWORD@postgres:5432/nolan?sslmode=disable

# Storage
RAW_INPUT_RETENTION_DAYS=30

# Control Plane
CONTROL_PLANE_PORT=8080
PRESIGNED_UPLOAD_TTL=15m
PRESIGNED_DOWNLOAD_TTL=1h

# Worker
WORKER_CONCURRENCY=2
MAX_JOB_RETRIES=3
RECLAIM_TIMEOUT_MS=300000

# Envoy
ENVOY_PORT=8443
"@

Set-Content -Path .env -Value $envContent -Encoding UTF8
Write-Host ".env generated successfully with secure random passwords."

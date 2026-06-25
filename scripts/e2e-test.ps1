param(
    [Parameter(Mandatory=$true)]
    [string]$VideoPath
)

# 1. Create a fake UUID for testing, or assume one exists
# First, insert a dummy project into Postgres if it doesn't exist
$DefaultProject = "00000000-0000-0000-0000-000000000000"
$DefaultUser = "00000000-0000-0000-0000-000000000000"
docker exec nolan-postgres-1 psql -U nolan -d nolan -c "INSERT INTO users (id, email, password_hash) VALUES ('$DefaultUser', 'test@test.com', 'hash') ON CONFLICT DO NOTHING;"
docker exec nolan-postgres-1 psql -U nolan -d nolan -c "INSERT INTO projects (id, user_id, name) VALUES ('$DefaultProject', '$DefaultUser', 'Test Project') ON CONFLICT DO NOTHING;"

$Ext = [System.IO.Path]::GetExtension($VideoPath)
$VideoId = [guid]::NewGuid().ToString()

Write-Host "1. Getting Upload URL for $VideoId$Ext..."
$UploadReq = @{ video_id = $VideoId; extension = $Ext } | ConvertTo-Json -Depth 5
$UploadRes = Invoke-RestMethod -Uri "http://localhost:8443/api/v1/upload-url" -Method Post -Body $UploadReq -ContentType "application/json"
$UploadUrl = $UploadRes.upload_url
$SourceKey = $UploadRes.source_key
Write-Host "   -> Source Key: $SourceKey"

Write-Host "2. Uploading $VideoPath to MinIO..."
# Use curl for binary upload since Invoke-RestMethod might corrupt binary files
curl.exe --resolve "minio:9000:127.0.0.1" -X PUT -T "$VideoPath" "$UploadUrl"

Write-Host "3. Creating Transcode Job..."
$JobReq = @{
    project_id = $DefaultProject
    video_id = $VideoId
    source_bucket = "raw-input"
    source_key = $SourceKey
    targets = @(
        @{ resolution = "480P"; container = "hls" },
        @{ resolution = "720P"; container = "hls" }
    )
} | ConvertTo-Json -Depth 5

$JobRes = Invoke-RestMethod -Uri "http://localhost:8443/api/v1/jobs" -Method Post -Body $JobReq -ContentType "application/json"
$JobId = $JobRes.id
Write-Host "   -> Job Enqueued! ID: $JobId"

Write-Host "4. Polling Job Status..."
$Status = "QUEUED"
while ($Status -eq "QUEUED" -or $Status -eq "PROCESSING") {
    Start-Sleep -Seconds 5
    $PollRes = Invoke-RestMethod -Uri "http://localhost:8443/api/v1/jobs/$JobId" -Method Get
    $Status = $PollRes.status
    Write-Host "   -> Status: $Status"
    
    if ($Status -eq "FAILED" -or $Status -eq "DLQ") {
        Write-Host "   -> ERROR: $($PollRes.error_message)"
        break
    }
}

if ($Status -eq "COMPLETED") {
    Write-Host ""
    Write-Host "SUCCESS! The video has been transcoded."
    Write-Host "You can stream the outputs via Envoy at:"
    Write-Host "  http://localhost:8443/media/$VideoId/hls/480p/master.m3u8"
    Write-Host "  http://localhost:8443/media/$VideoId/hls/720p/master.m3u8"
}

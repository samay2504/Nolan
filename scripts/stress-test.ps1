$Videos = @(
    "D:\Projects2.0\SmartAID\3.mp4",
    "D:\Projects2.0\SmartAID\4.mp4",
    "D:\Projects2.0\SmartAID\Demo.mp4",
    "D:\Projects2.0\SmartAID\hardware.mp4"
)

Write-Host "Starting Stress Test for 4 videos concurrently..." -ForegroundColor Cyan

$jobs = @()
foreach ($Video in $Videos) {
    Write-Host "Queueing $Video"
    $jobs += Start-Job -ScriptBlock {
        param($videoPath)
        Set-Location "D:\Projects2.0\System\Nolan"
        & .\scripts\e2e-test.ps1 -VideoPath $videoPath
    } -ArgumentList $Video
}

Write-Host "All videos queued! Waiting for jobs to complete..." -ForegroundColor Yellow

$completed = 0
while ($completed -lt $jobs.Count) {
    $completed = ($jobs | Where-Object { $_.State -eq 'Completed' -or $_.State -eq 'Failed' }).Count
    Write-Host "Completed: $completed / $($jobs.Count)"
    Start-Sleep -Seconds 10
}

Write-Host "`n--- Stress Test Results ---" -ForegroundColor Green
foreach ($job in $jobs) {
    Write-Host "`nResult for Job $($job.Id):" -ForegroundColor Cyan
    Receive-Job -Job $job
}

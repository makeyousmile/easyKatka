$ErrorActionPreference = "Stop"

$baseUrl = "https://api.opendota.com/api"
$timeoutSec = 20
$endpoints = @(
    "/",
    "/health",
    "/heroes",
    "/heroStats",
    "/metadata"
)

$hasFailures = $false

Write-Host "Проверка OpenDota API" -ForegroundColor Cyan
Write-Host ""

foreach ($endpoint in $endpoints) {
    $url = if ($endpoint -eq "/") { $baseUrl } else { $baseUrl + $endpoint }
    $statusCode = "ERROR"
    $durationMs = 0
    $details = ""

    $sw = [System.Diagnostics.Stopwatch]::StartNew()
    try {
        $response = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec $timeoutSec
        $statusCode = [string]$response.StatusCode
        $details = "OK"
    }
    catch {
        $exception = $_.Exception
        if ($exception.Response -and $exception.Response.StatusCode) {
            $statusCode = [int]$exception.Response.StatusCode
            $details = $exception.Message
        } else {
            $details = $exception.Message
        }
        $hasFailures = $true
    }
    finally {
        $sw.Stop()
        $durationMs = $sw.ElapsedMilliseconds
    }

    $line = "{0,-12} {1,-6} {2,6} ms  {3}" -f $endpoint, $statusCode, $durationMs, $details
    if ($statusCode -eq "200") {
        Write-Host $line -ForegroundColor Green
    } elseif ($statusCode -match '^\d+$') {
        Write-Host $line -ForegroundColor Yellow
    } else {
        Write-Host $line -ForegroundColor Red
    }
}

Write-Host ""
if ($hasFailures) {
    Write-Host "Есть недоступные или нестабильные endpoint'ы." -ForegroundColor Yellow
    exit 1
}

Write-Host "Все проверенные endpoint'ы ответили успешно." -ForegroundColor Green

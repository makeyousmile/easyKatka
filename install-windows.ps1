$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Test-Command {
    param([string]$Name)
    return $null -ne (Get-Command $Name -ErrorAction SilentlyContinue)
}

function Get-DockerDesktopPath {
    $candidates = @(
        "C:\Program Files\Docker\Docker\Docker Desktop.exe",
        "$env:LOCALAPPDATA\Programs\Docker\Docker\Docker Desktop.exe"
    )

    foreach ($candidate in $candidates) {
        if (Test-Path -LiteralPath $candidate) {
            return $candidate
        }
    }

    return $null
}

function Start-DockerDesktop {
    $dockerDesktop = Get-DockerDesktopPath
    if (-not $dockerDesktop) {
        throw "Docker Desktop установлен, но исполняемый файл не найден."
    }

    $isRunning = Get-Process -Name "Docker Desktop" -ErrorAction SilentlyContinue
    if (-not $isRunning) {
        Write-Step "Запускаю Docker Desktop"
        Start-Process -FilePath $dockerDesktop | Out-Null
    }
}

function Wait-DockerReady {
    param([int]$TimeoutSeconds = 180)

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        try {
            docker version | Out-Null
            docker compose version | Out-Null
            return
        }
        catch {
            Start-Sleep -Seconds 5
        }
    }

    throw "Docker не стал доступен за $TimeoutSeconds секунд."
}

function Install-DockerDesktop {
    if (-not (Test-Command "winget")) {
        throw "Не найден winget. Установите Docker Desktop вручную и запустите скрипт снова."
    }

    Write-Step "Устанавливаю Docker Desktop через winget"
    winget install -e --id Docker.DockerDesktop --accept-package-agreements --accept-source-agreements --silent
}

$projectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -LiteralPath $projectRoot

Write-Step "Проверяю Docker"
$dockerInstalled = Test-Command "docker"

if (-not $dockerInstalled) {
    Install-DockerDesktop
}

Start-DockerDesktop
Write-Step "Жду готовности Docker"
Wait-DockerReady

if (-not (Test-Path -LiteralPath ".env")) {
    if (-not (Test-Path -LiteralPath ".env.example")) {
        throw "Не найден .env и отсутствует .env.example."
    }

    Write-Step "Создаю .env из .env.example"
    Copy-Item -LiteralPath ".env.example" -Destination ".env"
}

$envContent = Get-Content -LiteralPath ".env" -Raw
if ($envContent -match "your_telegram_") {
    throw "Заполните .env реальными значениями TELEGRAM_BOT_TOKEN и TELEGRAM_NOTIFY_CHAT_ID, затем запустите install.ps1 снова."
}

if (-not (Test-Path -LiteralPath "account_id")) {
    throw "Не найден файл account_id."
}

Write-Step "Собираю и запускаю контейнер"
docker compose up -d --build

Write-Step "Готово"
Write-Host "Сервис easykatka запущен через Docker Compose." -ForegroundColor Green

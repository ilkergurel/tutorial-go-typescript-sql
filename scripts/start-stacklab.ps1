# Starts StackLab in local mode (ADR-0008) and opens it in the browser.
# The desktop shortcut runs this (scripts/create-desktop-shortcut.ps1 makes it);
# by hand: powershell -NoProfile -ExecutionPolicy Bypass -File scripts\start-stacklab.ps1
#
# The app runs in this window: close it, or press Ctrl+C, to stop the backend
# and Vite. Why a window and not a hidden background process: a visible window
# is the one obvious way to stop it, and its log shows why a start failed.

$repo = Split-Path -Parent $PSScriptRoot
Set-Location $repo
$Host.UI.RawUI.WindowTitle = 'StackLab - close this window to stop it'
$url = 'http://127.0.0.1:5173'

function Test-Listening([int]$Port) {
    [bool](Get-NetTCPConnection -LocalPort $Port -State Listen -ErrorAction SilentlyContinue)
}

function Stop-WithMessage([string]$Message) {
    Write-Host $Message -ForegroundColor Yellow
    Read-Host 'Press Enter to close this window'
    exit 1
}

# A second click while it runs just opens the page again.
if ((Test-Listening 8080) -and (Test-Listening 5173)) {
    Write-Host 'StackLab is already running; opening it in the browser.'
    Start-Process $url
    Start-Sleep -Seconds 3
    exit 0
}
if ((Test-Listening 8080) -or (Test-Listening 5173)) {
    Stop-WithMessage ('Port 8080 or 5173 is taken, probably by a StackLab that was not fully stopped. ' +
        'End stacklab.exe and node.exe in Task Manager, then click the shortcut again.')
}

# PostgreSQL is a Windows service set to start with Windows; starting it by hand needs admin rights.
$pg = Get-Service -Name 'postgresql*' -ErrorAction SilentlyContinue | Select-Object -First 1
if (-not $pg) {
    Stop-WithMessage 'No PostgreSQL service found. See README, "Local mode".'
}
if ($pg.Status -ne 'Running') {
    Stop-WithMessage ("PostgreSQL ($($pg.Name)) is not running. Start it from the Services app, " +
        "or in an admin PowerShell: Start-Service $($pg.Name)")
}
foreach ($tool in 'node', 'npm', 'go') {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        Stop-WithMessage "$tool is not on PATH. See README, `"Local mode`"."
    }
}

# npm start keeps this window busy, so a hidden helper opens the browser once the
# app answers. /healthz goes through Vite's proxy to the backend, so a 200 means
# both are up (the backend is built first, which can take a while after a Go update).
$opener = "for (`$i = 0; `$i -lt 300; `$i++) { " +
    "try { Invoke-WebRequest -UseBasicParsing -TimeoutSec 2 '$url/healthz' | Out-Null; Start-Process '$url'; break } " +
    "catch { Start-Sleep -Seconds 1 } }"
Start-Process powershell.exe -WindowStyle Hidden -ArgumentList '-NoProfile', '-Command', $opener

Write-Host 'Starting StackLab; the browser opens when it is ready.' -ForegroundColor Cyan
Write-Host 'Close this window (or press Ctrl+C) to stop it.' -ForegroundColor Cyan
Write-Host ''
npm start
if ($LASTEXITCODE -ne 0) {
    Read-Host "StackLab stopped (exit code $LASTEXITCODE). Press Enter to close this window"
}

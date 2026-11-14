# Creates (or replaces) a "StackLab" shortcut on the desktop that runs
# scripts/start-stacklab.ps1. Run it once from the repo root:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts\create-desktop-shortcut.ps1
# Run it again after moving the repo: the shortcut stores absolute paths.

$repo = Split-Path -Parent $PSScriptRoot
$launcher = Join-Path $PSScriptRoot 'start-stacklab.ps1'
# GetFolderPath follows a desktop that OneDrive has moved.
$desktop = [Environment]::GetFolderPath('Desktop')
$link = Join-Path $desktop 'StackLab.lnk'

$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($link)
$shortcut.TargetPath = Join-Path $env:SystemRoot 'System32\WindowsPowerShell\v1.0\powershell.exe'
# -ExecutionPolicy Bypass applies to this one process only, so the machine's policy stays as it is.
$shortcut.Arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$launcher`""
$shortcut.WorkingDirectory = $repo
$shortcut.Description = 'Start StackLab (backend + Vite) and open it in the browser'
$shortcut.IconLocation = (Join-Path $env:SystemRoot 'System32\shell32.dll') + ',13'
$shortcut.Save()

Write-Host "Created $link"

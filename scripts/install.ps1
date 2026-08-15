param(
    [string]$InstallRoot = (Join-Path $env:LOCALAPPDATA 'Programs\VRC++'),
    [switch]$DesktopShortcut,
    [switch]$Startup
)

$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$bundledExe = Join-Path $PSScriptRoot 'vrc-plus-plus.exe'
$developmentExe = Join-Path $workspace 'dist\vrc-plus-plus.exe'
$sourceExe = if (Test-Path -LiteralPath $bundledExe) { $bundledExe } else { $developmentExe }
if (-not (Test-Path -LiteralPath $sourceExe)) {
    throw 'VRC++ executable was not found next to the installer or in dist.'
}

$resolvedParent = [System.IO.Path]::GetFullPath((Split-Path -Parent $InstallRoot))
$allowedParent = [System.IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs'))
if (-not $resolvedParent.StartsWith($allowedParent, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "InstallRoot must remain under $allowedParent"
}

New-Item -ItemType Directory -Path $InstallRoot -Force | Out-Null
$installedExe = Join-Path $InstallRoot 'vrc-plus-plus.exe'
Copy-Item -LiteralPath $sourceExe -Destination $installedExe -Force
Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'uninstall.ps1') -Destination (Join-Path $InstallRoot 'uninstall.ps1') -Force

$shell = New-Object -ComObject WScript.Shell
function New-HarborShortcut([string]$ShortcutPath) {
    $shortcut = $shell.CreateShortcut($ShortcutPath)
    $shortcut.TargetPath = $installedExe
    $shortcut.WorkingDirectory = $InstallRoot
    $shortcut.Description = 'VRC++ local VRChat companion'
    $shortcut.Save()
}

$startMenuDirectory = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\VRC++'
New-Item -ItemType Directory -Path $startMenuDirectory -Force | Out-Null
New-HarborShortcut (Join-Path $startMenuDirectory 'VRC++.lnk')

if ($DesktopShortcut) {
    New-HarborShortcut (Join-Path ([Environment]::GetFolderPath('Desktop')) 'VRC++.lnk')
}
if ($Startup) {
    New-HarborShortcut (Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Startup\VRC++.lnk')
}

Write-Host "VRC++ installed to $InstallRoot"
Write-Host 'User data remains in the configured VRC++ data directory.'
Start-Process -FilePath $installedExe

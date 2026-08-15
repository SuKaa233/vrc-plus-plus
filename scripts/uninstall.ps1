param(
    [string]$InstallRoot = (Join-Path $env:LOCALAPPDATA 'Programs\VRC++')
)

$ErrorActionPreference = 'Stop'
$resolvedTarget = [System.IO.Path]::GetFullPath($InstallRoot).TrimEnd('\')
$allowedParent = [System.IO.Path]::GetFullPath((Join-Path $env:LOCALAPPDATA 'Programs')).TrimEnd('\')
if ($resolvedTarget -eq $allowedParent -or -not $resolvedTarget.StartsWith($allowedParent + '\', [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to remove path outside $allowedParent"
}

Get-Process -Name 'vrc-plus-plus' -ErrorAction SilentlyContinue | Stop-Process -Force
$shortcuts = @(
    (Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\VRC++'),
    (Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Startup\VRC++.lnk'),
    (Join-Path ([Environment]::GetFolderPath('Desktop')) 'VRC++.lnk')
)
foreach ($shortcut in $shortcuts) {
    if (Test-Path -LiteralPath $shortcut) { Remove-Item -LiteralPath $shortcut -Recurse -Force }
}
if (Test-Path -LiteralPath $resolvedTarget) {
    Remove-Item -LiteralPath $resolvedTarget -Recurse -Force
}
Write-Host 'VRC++ application files were removed. Local user data was preserved.'

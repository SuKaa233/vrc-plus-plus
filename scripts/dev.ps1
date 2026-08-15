param(
    [int]$Port = 47831
)

$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$goExe = Join-Path $workspace '.tools\go\bin\go.exe'

if (-not (Test-Path -LiteralPath $goExe)) {
    throw 'Project Go toolchain is missing. See docs/development.md.'
}

Push-Location (Join-Path $workspace 'apps\gateway')
try {
    & $goExe run '.\cmd\gateway' -listen "127.0.0.1:$Port" -dev-origin 'http://127.0.0.1:5173' -data-dir (Join-Path $workspace '.data') -open-browser=false
} finally {
    Pop-Location
}

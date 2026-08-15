param(
    [switch]$SkipTests
)

$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$goExe = Join-Path $workspace '.tools\go\bin\go.exe'
$releaseVersion = (Get-Content -LiteralPath (Join-Path $workspace 'package.json') -Raw | ConvertFrom-Json).version

if (-not (Test-Path -LiteralPath $goExe)) {
    throw 'Project Go toolchain is missing. See docs/development.md.'
}

Push-Location $workspace
try {
    $embeddedDocs = @(
        'feature-roadmap.md',
        'ideas-backlog.md',
        'implementation-blueprint.md',
        'implementation-status.md',
        'vrchat-api-integration.md',
        'windows-release.md'
    )
    foreach ($document in $embeddedDocs) {
        Copy-Item -LiteralPath (Join-Path $workspace "docs\$document") -Destination (Join-Path $workspace "apps\gateway\docs\$document") -Force
    }

    & npm.cmd --prefix apps/web run build
    if ($LASTEXITCODE -ne 0) { throw 'Web build failed.' }

    if (-not $SkipTests) {
        & npm.cmd --prefix apps/web run test
        if ($LASTEXITCODE -ne 0) { throw 'Web tests failed.' }
        Push-Location 'apps\gateway'
        try {
            & $goExe test ./...
            if ($LASTEXITCODE -ne 0) { throw 'Gateway tests failed.' }
        } finally {
            Pop-Location
        }
    }

    New-Item -ItemType Directory -Path 'dist' -Force | Out-Null
    Push-Location 'apps\gateway'
    try {
        & $goExe build -trimpath -ldflags="-s -w -X main.version=$releaseVersion" -o '..\..\dist\vrc-plus-plus.exe' '.\cmd\gateway'
        if ($LASTEXITCODE -ne 0) { throw 'Gateway build failed.' }
    } finally {
        Pop-Location
    }
} finally {
    Pop-Location
}

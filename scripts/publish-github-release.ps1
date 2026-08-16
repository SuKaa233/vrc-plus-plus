param(
    [string]$Repository = 'SuKaa233/vrc-plus-plus',
    [switch]$Draft
)

$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$version = (Get-Content -LiteralPath (Join-Path $workspace 'package.json') -Raw | ConvertFrom-Json).version
$tag = "v$version"
$installer = Join-Path $workspace "dist\VRC++-Setup-$version.exe"
$manifest = Join-Path $workspace 'dist\update-manifest.json'
foreach ($file in @($installer, $manifest)) {
    if (-not (Test-Path -LiteralPath $file)) { throw "Release file is missing: $file" }
}
$gh = Get-Command gh.exe -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -First 1
if (-not $gh) {
    $ghCandidates = @(
        (Join-Path $env:ProgramFiles 'GitHub CLI\gh.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\GitHub CLI\gh.exe')
    ) | Where-Object { Test-Path -LiteralPath $_ }
    $gh = $ghCandidates | Select-Object -First 1
}
if (-not $gh) {
    throw 'GitHub CLI gh.exe is required. Install it, then run gh auth login.'
}
& $gh auth status
if ($LASTEXITCODE -ne 0) { throw 'GitHub CLI is not authenticated. Run gh auth login.' }

$releaseNotes = Join-Path $workspace "docs\release-notes-$version.md"
$releaseArgs = @('release', 'create', $tag, $installer, $manifest, '--repo', $Repository, '--title', "VRC++ $version")
if (Test-Path -LiteralPath $releaseNotes) {
    $releaseArgs += @('--notes-file', $releaseNotes)
} else {
    $releaseArgs += '--generate-notes'
}
if ($Draft) { $releaseArgs += '--draft' }
if ($version.Contains('-')) { $releaseArgs += '--prerelease' }
& $gh @releaseArgs
if ($LASTEXITCODE -ne 0) {
    & $gh release upload $tag $installer $manifest --repo $Repository --clobber
    if ($LASTEXITCODE -ne 0) { throw "Could not create or update release $tag." }
}

if ($version.Contains('-')) {
    & $gh release view update-beta --repo $Repository 2>$null
    if ($LASTEXITCODE -ne 0) {
        & $gh release create update-beta $manifest --repo $Repository --title 'VRC++ Beta Update Channel' --notes 'This rolling release stores the Beta update manifest.' --prerelease
    } else {
        & $gh release upload update-beta $manifest --repo $Repository --clobber
    }
    if ($LASTEXITCODE -ne 0) { throw 'Could not update the Beta channel manifest.' }
} else {
    & $gh release upload $tag $manifest --repo $Repository --clobber
    if ($LASTEXITCODE -ne 0) { throw 'Could not upload the stable update manifest.' }
}

Write-Host "Published VRC++ $version to https://github.com/$Repository/releases/tag/$tag"

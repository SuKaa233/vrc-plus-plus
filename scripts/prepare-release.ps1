param(
    [string[]]$ReleaseNotes = @(),
    [string]$PrimaryDownloadBaseUrl,
    [string]$Repository = 'SuKaa233/vrc-plus-plus'
)

$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
$version = (Get-Content -LiteralPath (Join-Path $workspace 'package.json') -Raw | ConvertFrom-Json).version
$installerName = "VRC++-Setup-$version.exe"
$installer = Join-Path $workspace "dist\$installerName"
if (-not (Test-Path -LiteralPath $installer)) {
    throw "Installer was not found. Run scripts\package.ps1 first: $installer"
}

$mirrors = [System.Collections.Generic.List[string]]::new()
if ($PrimaryDownloadBaseUrl) {
    $mirrors.Add($PrimaryDownloadBaseUrl.TrimEnd('/') + '/' + $installerName)
}
$mirrors.Add("https://github.com/$Repository/releases/download/v$version/$installerName")
if ($ReleaseNotes.Count -eq 0) {
    $releaseNotesFile = Join-Path $workspace "docs\release-notes-$version.md"
    if (Test-Path -LiteralPath $releaseNotesFile) {
        $ReleaseNotes = @(Get-Content -LiteralPath $releaseNotesFile -Encoding UTF8 | Where-Object { $_.TrimStart().StartsWith('- ') } | ForEach-Object { $_.Trim().Substring(2) } | Select-Object -First 4)
    }
    if ($ReleaseNotes.Count -eq 0) { $ReleaseNotes = @("Update VRC++ to $version") }
}

$manifest = [ordered]@{
    version = $version
    publishedAt = [DateTimeOffset]::UtcNow.ToString('o')
    releaseNotes = @($ReleaseNotes)
    windowsX64 = [ordered]@{
        file = $installerName
        size = (Get-Item -LiteralPath $installer).Length
        mirrors = @($mirrors)
    }
}
$output = Join-Path $workspace 'dist\update-manifest.json'
$json = $manifest | ConvertTo-Json -Depth 6
[IO.File]::WriteAllText($output, $json + [Environment]::NewLine, [Text.UTF8Encoding]::new($false))
Write-Host "Update manifest: $output"

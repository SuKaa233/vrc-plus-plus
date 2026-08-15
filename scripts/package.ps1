param([switch]$SkipTests, [switch]$RequireSigned)

$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
Push-Location $workspace
try {
    & (Join-Path $PSScriptRoot 'build.ps1') -SkipTests:$SkipTests
    if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }

    & (Join-Path $PSScriptRoot 'sign.ps1') -Binary 'dist\vrc-plus-plus.exe' -RequireSigned:$RequireSigned

    $version = (Get-Content -LiteralPath (Join-Path $workspace 'package.json') -Raw | ConvertFrom-Json).version
    $stage = Join-Path $workspace "dist\package\vrc-plus-plus-$version-windows-x64"
    $resolvedWorkspace = [IO.Path]::GetFullPath($workspace).TrimEnd('\') + '\'
    $resolvedStage = [IO.Path]::GetFullPath($stage)
    if (-not $resolvedStage.StartsWith($resolvedWorkspace, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Unsafe package staging path: $resolvedStage"
    }
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force }
    New-Item -ItemType Directory -Path $stage -Force | Out-Null
    Copy-Item -LiteralPath 'dist\vrc-plus-plus.exe' -Destination $stage
    Copy-Item -LiteralPath 'scripts\install.ps1' -Destination $stage
    Copy-Item -LiteralPath 'scripts\uninstall.ps1' -Destination $stage
    Copy-Item -LiteralPath 'README.md' -Destination $stage

    $archive = Join-Path $workspace "dist\vrc-plus-plus-$version-windows-x64.zip"
    if (Test-Path -LiteralPath $archive) { Remove-Item -LiteralPath $archive -Force }
    Compress-Archive -LiteralPath $stage -DestinationPath $archive -CompressionLevel Optimal
    $hash = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    $manifest = [ordered]@{
        version = $version
        publishedAt = (Get-Date).ToUniversalTime().ToString('o')
        windowsX64 = [ordered]@{
            file = (Split-Path -Leaf $archive)
            sha256 = $hash
            size = (Get-Item -LiteralPath $archive).Length
            mirrors = @()
        }
    }
    $manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath 'dist\update-manifest.json' -Encoding UTF8
    Write-Host "Package: $archive"
    Write-Host "SHA256: $hash"
} finally {
    Pop-Location
}

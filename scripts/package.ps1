param(
    [switch]$SkipTests,
    [switch]$RequireSigned,
    [string]$InnoCompiler
)

$ErrorActionPreference = 'Stop'
$workspace = Split-Path -Parent $PSScriptRoot
Push-Location $workspace
try {
    & (Join-Path $PSScriptRoot 'build.ps1') -SkipTests:$SkipTests
    if ($LASTEXITCODE -ne 0) { throw 'Build failed.' }

    $version = (Get-Content -LiteralPath (Join-Path $workspace 'package.json') -Raw | ConvertFrom-Json).version
    & (Join-Path $PSScriptRoot 'sign.ps1') -Binary 'dist\vrc-plus-plus.exe' -RequireSigned:$RequireSigned

    $compilerCandidates = @(
        $InnoCompiler,
        $env:VRC_PLUS_PLUS_ISCC,
        (Join-Path $workspace '.tools\inno\ISCC.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\Inno Setup 7\ISCC.exe'),
        (Join-Path $env:ProgramFiles 'Inno Setup 7\ISCC.exe'),
        (Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6\ISCC.exe')
    ) | Where-Object { $_ -and (Test-Path -LiteralPath $_) }
    $iscc = $compilerCandidates | Select-Object -First 1
    if (-not $iscc) {
        throw 'Inno Setup compiler was not found. Install Inno Setup 7 or pass -InnoCompiler <path-to-ISCC.exe>.'
    }

    $sourceExe = [IO.Path]::GetFullPath((Join-Path $workspace 'dist\vrc-plus-plus.exe'))
    $outputDirectory = [IO.Path]::GetFullPath((Join-Path $workspace 'dist'))
    $installerScript = [IO.Path]::GetFullPath((Join-Path $workspace 'installer\VRCPlusPlus.iss'))
    $versionMatch = [regex]::Match($version, '^(\d+)\.(\d+)\.(\d+)(?:-beta\.(\d+))?$')
    if (-not $versionMatch.Success) { throw "Unsupported installer version format: $version" }
    $revision = if ($versionMatch.Groups[4].Success) { $versionMatch.Groups[4].Value } else { '0' }
    $numericVersion = '{0}.{1}.{2}.{3}' -f $versionMatch.Groups[1].Value, $versionMatch.Groups[2].Value, $versionMatch.Groups[3].Value, $revision
    & $iscc "/DAppVersion=$version" "/DAppNumericVersion=$numericVersion" "/DSourceExe=$sourceExe" "/DOutputDirectory=$outputDirectory" $installerScript
    if ($LASTEXITCODE -ne 0) { throw 'Inno Setup compilation failed.' }

    $installer = Join-Path $outputDirectory "VRC++-Setup-$version.exe"
    if (-not (Test-Path -LiteralPath $installer)) { throw "Installer was not produced: $installer" }
    & (Join-Path $PSScriptRoot 'sign.ps1') -Binary $installer -RequireSigned:$RequireSigned
    & (Join-Path $PSScriptRoot 'prepare-release.ps1')
    Write-Host "Installer: $installer"
} finally {
    Pop-Location
}

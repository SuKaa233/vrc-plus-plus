param(
    [Parameter(Mandatory = $true)][string]$Binary,
    [string]$CertificateThumbprint = $env:VRC_HARBOR_SIGN_CERT_THUMBPRINT,
    [string]$CertificatePath = $env:VRC_HARBOR_SIGN_CERT_PATH,
    [switch]$RequireSigned
)

$ErrorActionPreference = 'Stop'
$resolvedBinary = (Resolve-Path -LiteralPath $Binary).Path
$signtool = Get-ChildItem "${env:ProgramFiles(x86)}\Windows Kits\10\bin" -Filter signtool.exe -Recurse -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -match '\\x64\\signtool\.exe$' } |
    Sort-Object FullName -Descending |
    Select-Object -First 1 -ExpandProperty FullName

if (-not $signtool) {
    if ($RequireSigned) { throw 'Windows SDK signtool.exe is required for a signed release.' }
    Write-Warning 'signtool.exe was not found; leaving the local development binary unsigned.'
    return
}

$certificateArgs = @()
if ($CertificateThumbprint) {
    $certificateArgs = @('/sha1', $CertificateThumbprint, '/sm')
} elseif ($CertificatePath) {
    $resolvedCertificate = (Resolve-Path -LiteralPath $CertificatePath).Path
    $certificateArgs = @('/f', $resolvedCertificate)
    if ($env:VRC_HARBOR_SIGN_CERT_PASSWORD) { $certificateArgs += @('/p', $env:VRC_HARBOR_SIGN_CERT_PASSWORD) }
} else {
    if ($RequireSigned) { throw 'Set VRC_HARBOR_SIGN_CERT_THUMBPRINT or VRC_HARBOR_SIGN_CERT_PATH.' }
    Write-Warning 'No signing certificate is configured; leaving the local development binary unsigned.'
    return
}

$timestampServers = @(
    'http://timestamp.digicert.com',
    'http://timestamp.sectigo.com'
)
$signed = $false
foreach ($server in $timestampServers) {
    & $signtool sign /fd SHA256 /tr $server /td SHA256 @certificateArgs $resolvedBinary
    if ($LASTEXITCODE -eq 0) { $signed = $true; break }
}
if (-not $signed) { throw 'Authenticode signing failed against every configured timestamp source.' }

& $signtool verify /pa /all $resolvedBinary
if ($LASTEXITCODE -ne 0) { throw 'Authenticode verification failed.' }
Write-Host "Signed and verified: $resolvedBinary"

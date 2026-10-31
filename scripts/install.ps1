<#
.SYNOPSIS
  Installs or updates tgexport on Windows.

.DESCRIPTION
  Downloads the latest (or the requested) tgexport release from GitHub,
  verifies its SHA-256 checksum and installs it to
  %LOCALAPPDATA%\Programs\tgexport, adding that folder to the user PATH.
  Running it again updates tgexport, or does nothing if it is up to date.

  irm https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/install.ps1 | iex

  With parameters:
  & ([scriptblock]::Create((irm https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/install.ps1))) -Version v0.1.0 -Force

.PARAMETER Version
  Release tag to install, e.g. v0.1.0. Defaults to the latest release.

.PARAMETER InstallDir
  Install directory. Defaults to %LOCALAPPDATA%\Programs\tgexport.

.PARAMETER Force
  Reinstall even if the same version is already installed.
#>
[CmdletBinding()]
param(
    [string]$Version = $env:TGEXPORT_VERSION,
    [string]$InstallDir = $(if ($env:TGEXPORT_INSTALL_DIR) { $env:TGEXPORT_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\tgexport' }),
    [switch]$Force
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'  # Invoke-WebRequest is very slow with the progress bar
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Repo = 'antalkon/tgexport'
$Exe = 'tgexport.exe'

switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { $Arch = 'amd64' }
    'ARM64' { $Arch = 'arm64' }
    default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

if (-not $Version) {
    $release = Invoke-RestMethod -UseBasicParsing "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ 'User-Agent' = 'tgexport-installer' }
    $Version = $release.tag_name
}
if (-not $Version.StartsWith('v')) { $Version = "v$Version" }

$Target = Join-Path $InstallDir $Exe
$Current = $null
if (Test-Path $Target) {
    # "tgexport 0.1.0 (commit ...)" -> "0.1.0"
    $Current = ((& $Target version) -split ' ')[1] -replace '^v', ''
}
if ($Current -eq $Version.TrimStart('v') -and -not $Force) {
    Write-Host "tgexport $Version is already installed at $Target - nothing to do. Use -Force to reinstall."
    return
}

$Archive = "tgexport_windows_$Arch.zip"
$Base = "https://github.com/$Repo/releases/download/$Version"
$Tmp = Join-Path ([IO.Path]::GetTempPath()) ("tgexport-" + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $Tmp | Out-Null

try {
    if ($Current) { Write-Host "Updating tgexport $Current -> $($Version.TrimStart('v')) (windows/$Arch)..." }
    else { Write-Host "Installing tgexport $($Version.TrimStart('v')) (windows/$Arch)..." }

    Invoke-WebRequest -UseBasicParsing "$Base/$Archive" -OutFile (Join-Path $Tmp $Archive)
    Invoke-WebRequest -UseBasicParsing "$Base/checksums.txt" -OutFile (Join-Path $Tmp 'checksums.txt')

    $line = Get-Content (Join-Path $Tmp 'checksums.txt') | Where-Object { ($_ -split '\s+')[1] -eq $Archive }
    if (-not $line) { throw "$Archive is not listed in checksums.txt" }
    $expected = ($line -split '\s+')[0].ToLower()
    $actual = (Get-FileHash (Join-Path $Tmp $Archive) -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) { throw "Checksum mismatch for $Archive (expected $expected, got $actual)" }

    Expand-Archive -Path (Join-Path $Tmp $Archive) -DestinationPath $Tmp -Force
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item (Join-Path $Tmp $Exe) $Target -Force
}
finally {
    Remove-Item $Tmp -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host "[ok] $(& $Target version)"
Write-Host "  installed to $Target"

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$parts = @($userPath -split ';' | Where-Object { $_ })
if ($parts -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable('Path', (($parts + $InstallDir) -join ';'), 'User')
    $env:Path = "$env:Path;$InstallDir"
    Write-Host "  added $InstallDir to your user PATH (open a new terminal to pick it up)"
}

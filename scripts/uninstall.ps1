<#
.SYNOPSIS
  Removes tgexport from Windows.

.DESCRIPTION
  Deletes the install directory created by install.ps1 and removes it from
  the user PATH.

  irm https://raw.githubusercontent.com/antalkon/tgexport/main/scripts/uninstall.ps1 | iex
#>
[CmdletBinding()]
param(
    [string]$InstallDir = $(if ($env:TGEXPORT_INSTALL_DIR) { $env:TGEXPORT_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\tgexport' })
)

$ErrorActionPreference = 'Stop'

$removed = $false
if (Test-Path $InstallDir) {
    Remove-Item $InstallDir -Recurse -Force
    Write-Host "[ok] removed $InstallDir"
    $removed = $true
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$parts = @($userPath -split ';' | Where-Object { $_ })
if ($parts -contains $InstallDir) {
    [Environment]::SetEnvironmentVariable('Path', (($parts | Where-Object { $_ -ne $InstallDir }) -join ';'), 'User')
    Write-Host "[ok] removed $InstallDir from user PATH"
    $removed = $true
}

if ($removed) { Write-Host 'tgexport has been uninstalled.' }
else { Write-Host 'tgexport is not installed - nothing to remove.' }

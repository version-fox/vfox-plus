# Copyright 2026 Han Li and contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

<#
.SYNOPSIS
  Self-sign the vfox+ MSIX packages and install the bundle on Windows 11.

.DESCRIPTION
  One command that replaces the manual signing flow: it creates (or reuses) a
  self-signed code-signing certificate for CN=<PublisherCN>, rebuilds the
  .msix/.msixbundle with the matching publisher via vfxtool build-msix,
  signs them with osslsigncode (downloaded automatically, no Windows SDK
  needed), trusts the certificate for the local machine and installs the
  bundle. No third-party certificate is needed; see
  shared/packaging/windows/README.md for the trade-offs.

  Run from the extracted windows artifact directory (it holds stage-x64/,
  stage-arm64/, vfxtool.exe and this script):

    powershell -ExecutionPolicy Bypass -File Sign-Msix.ps1

  The script relaunches itself elevated: installing the certificate and the
  package both need administrator rights.
#>
[CmdletBinding()]
param(
  # Directory holding stage-x64/, stage-arm64/, vfxtool.exe and this script.
  # Defaults to the script's own directory.
  [string]$Directory = (Split-Path -Parent $MyInvocation.MyCommand.Path),

  # Common name of the self-signed certificate and of the package publisher.
  # Must match ^[A-Za-z0-9][A-Za-z0-9 ._-]*$ (no commas): it becomes
  # Publisher="CN=<PublisherCN>" of the signed packages.
  [string]$PublisherCN = "vfox-plus",

  # Package file name prefix, as passed to vfxtool build-msix --package.
  [string]$Package = "vfox-plus",

  # Application version, e.g. 0.1.0. Empty means auto-detect from the
  # unsigned <Package>_<version>.msixbundle in -Directory.
  [string]$Version = "",

  # Rebuild and reinstall even when the signed bundle is up to date, and
  # create a fresh certificate instead of reusing the existing one.
  [switch]$Force,

  # Rebuild and sign but do not install the bundle.
  [switch]$SkipInstall
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# osslsigncode signs APPX/MSIX on any OS; pinned so every run uses the same
# tool. No Windows SDK (SignTool) is needed.
$OsslsigncodeVersion = "2.14"
$OsslsigncodeSha256 = "9a1722aaf62a27852c4eb9c35749a0248065052d0ae0a93d4ed6bb49def027f2"
$OsslsigncodeUrl = "https://github.com/mtrojnar/osslsigncode/releases/download/$OsslsigncodeVersion/osslsigncode-$OsslsigncodeVersion-windows-x64-mingw.zip"

function Test-Administrator {
  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = New-Object Security.Principal.WindowsPrincipal($identity)
  return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Administrator)) {
  Write-Host "Relaunching elevated (certificate trust and install need administrator rights)..."
  $args = @("-ExecutionPolicy", "Bypass", "-File", $MyInvocation.MyCommand.Path,
    "-Directory", $Directory, "-PublisherCN", $PublisherCN, "-Package", $Package)
  if ($Version) { $args += @("-Version", $Version) }
  if ($Force) { $args += "-Force" }
  if ($SkipInstall) { $args += "-SkipInstall" }
  $proc = Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -PassThru -Wait
  exit $proc.ExitCode
}

if ($PublisherCN -notmatch '^[A-Za-z0-9][A-Za-z0-9 ._-]*$') {
  throw "PublisherCN must look like a common name without commas: $PublisherCN"
}
$Directory = (Resolve-Path $Directory).Path
$stageX64 = Join-Path $Directory "stage-x64"
$stageArm64 = Join-Path $Directory "stage-arm64"
$vfxtool = Join-Path $Directory "vfxtool.exe"
foreach ($need in @($vfxtool)) {
  if (-not (Test-Path $need)) { throw "Missing $need: run this script from the extracted windows artifact directory." }
}
$stages = @()
if (Test-Path (Join-Path $stageX64 "vfox-plus.exe")) { $stages += "x64" }
if (Test-Path (Join-Path $stageArm64 "vfox-plus.exe")) { $stages += "arm64" }
if ($stages.Count -eq 0) { throw "No vfox-plus.exe in $stageX64 or $stageArm64." }

if (-not $Version) {
  $bundle = Get-ChildItem -Path $Directory -Filter "$Package`_*.msixbundle" | Select-Object -First 1
  if (-not $bundle) { throw "No $Package`_*.msixbundle in $Directory: pass -Version explicitly." }
  if ($bundle.Name -notmatch '^.+_(\d+\.\d+\.\d+)\.msixbundle$') {
    throw "Cannot parse a version from $($bundle.Name): pass -Version explicitly."
  }
  $Version = $Matches[1]
}
Write-Host "Version: $Version   Architectures: $($stages -join ', ')   Publisher: CN=$PublisherCN"

# 1. Signing tool.
$toolsDir = Join-Path $Directory "tools"
$ossl = Get-ChildItem -Path $toolsDir -Recurse -Filter "osslsigncode.exe" -ErrorAction SilentlyContinue |
  Select-Object -First 1 -ExpandProperty FullName
if ((-not $ossl) -or $Force) {
  Write-Host "Downloading osslsigncode $OsslsigncodeVersion..."
  New-Item -ItemType Directory -Force -Path $toolsDir | Out-Null
  $zipPath = Join-Path $toolsDir "osslsigncode.zip"
  Invoke-WebRequest -Uri $OsslsigncodeUrl -OutFile $zipPath
  $hash = (Get-FileHash -Path $zipPath -Algorithm SHA256).Hash.ToLower()
  if ($hash -ne $OsslsigncodeSha256) {
    throw "osslsigncode checksum mismatch: got $hash"
  }
  Expand-Archive -Path $zipPath -DestinationPath (Join-Path $toolsDir "osslsigncode") -Force
  Remove-Item $zipPath
  $ossl = Get-ChildItem -Path $toolsDir -Recurse -Filter "osslsigncode.exe" |
    Select-Object -First 1 -ExpandProperty FullName
  if (-not $ossl) { throw "osslsigncode.exe not found after extracting." }
}
Write-Host "Using $ossl"

# 2. Self-signed code-signing certificate (CurrentUser\My, exportable key).
$ekuCodeSigning = "1.3.6.1.5.5.7.3.3"
$existing = Get-ChildItem "Cert:\CurrentUser\My" -ErrorAction SilentlyContinue |
  Where-Object {
    $_.Subject -eq "CN=$PublisherCN" -and $_.HasPrivateKey -and $_.NotAfter -gt (Get-Date) -and
    ($_.EnhancedKeyUsageList | Where-Object { $_.ObjectId.Value -eq $ekuCodeSigning })
  } |
  Sort-Object NotAfter -Descending | Select-Object -First 1
if ($existing -and -not $Force) {
  $cert = $existing
  Write-Host "Reusing certificate $($cert.Thumbprint) (expires $($cert.NotAfter.ToString('yyyy-MM-dd')))."
} else {
  Write-Host "Creating a self-signed code-signing certificate for CN=$PublisherCN..."
  $cert = New-SelfSignedCertificate -Type Custom -Subject "CN=$PublisherCN" `
    -KeyUsage DigitalSignature `
    -TextExtension @("2.5.29.37={text}$ekuCodeSigning") `
    -KeyAlgorithm RSA -KeyLength 2048 `
    -NotAfter (Get-Date).AddYears(5) `
    -CertStoreLocation "Cert:\CurrentUser\My"
}

# 3. Trust it for this machine (sideloading trusts TrustedPeople).
$trusted = Get-ChildItem "Cert:\LocalMachine\TrustedPeople" -ErrorAction SilentlyContinue |
  Where-Object { $_.Thumbprint -eq $cert.Thumbprint }
if (-not $trusted) {
  Write-Host "Trusting the certificate for the local machine (TrustedPeople)..."
  $store = New-Object System.Security.Cryptography.X509Certificates.X509Store("TrustedPeople", "LocalMachine")
  $store.Open("ReadWrite")
  try { $store.Add($cert) } finally { $store.Close() }
} else {
  Write-Host "Certificate already trusted."
}

# 4. Rebuild the packages with the matching publisher (signed packages carry
# no OID; the unsigned Publisher with the OID would not match this cert).
$signedDir = Join-Path $Directory "signed"
New-Item -ItemType Directory -Force -Path $signedDir | Out-Null
$buildArgs = @("build-msix", "--package", $Package, "--version", $Version,
  "--identity", "dev.vfox.VfoxPlus", "--publisher", "CN=$PublisherCN",
  "--executable", "vfox-plus.exe")
if ($stages -contains "x64") { $buildArgs += @("--stage-x64", $stageX64) }
if ($stages -contains "arm64") { $buildArgs += @("--stage-arm64", $stageArm64) }
$buildArgs += $signedDir
Write-Host "Rebuilding signed packages..."
& $vfxtool @buildArgs
if ($LASTEXITCODE -ne 0) { throw "vfxtool build-msix failed." }

# 5. Sign every .msix first, then the bundle.
$pfxPassword = -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 32 | ForEach-Object { [char]$_ })
$pfxPath = Join-Path ([System.IO.Path]::GetTempPath()) "vfox-plus-signing-$([Guid]::NewGuid().ToString('N')).pfx"
$securePassword = ConvertTo-SecureString -String $pfxPassword -AsPlainText -Force
Export-PfxCertificate -Cert $cert -FilePath $pfxPath -Password $securePassword | Out-Null
try {
  $toSign = @(Get-ChildItem -Path $signedDir -Filter "*.msix") +
    @(Get-ChildItem -Path $signedDir -Filter "*.msixbundle")
  foreach ($file in $toSign) {
    Write-Host "Signing $($file.Name)..."
    $tmp = "$($file.FullName).signed-tmp"
    & $ossl sign -pkcs12 $pfxPath -pass $pfxPassword -in $file.FullName -out $tmp
    if ($LASTEXITCODE -ne 0) { throw "osslsigncode failed for $($file.Name)." }
    Move-Item -Force -Path $tmp -Destination $file.FullName
  }
} finally {
  Remove-Item -Force -ErrorAction SilentlyContinue $pfxPath
}

$signedBundle = Get-ChildItem -Path $signedDir -Filter "$Package`_$Version.msixbundle" |
  Select-Object -First 1
if (-not $signedBundle) { throw "Signed bundle not found in $signedDir." }

if (-not $SkipInstall) {
  Write-Host "Installing $($signedBundle.Name)..."
  Add-AppxPackage -Path $signedBundle.FullName
  Write-Host "Installed. Launch vfox+ from the Start menu."
} else {
  Write-Host "Signed bundle: $($signedBundle.FullName)"
}

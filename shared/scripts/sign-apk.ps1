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

# Signs the APK of the phone app with a key. release.yml runs it with the release key, and the
# `signing` job of phone.yml runs it with a key that is made for the run and is no one's, on every
# change: the two run the same steps, so what the rehearsal passes is what the release runs, and only
# the key differs.
#
# The APK that mygo builds is signed with the debug key. zipalign aligns it again, with the 16 KB pages
# of its native libraries (-P cannot be combined with -p), and apksigner replaces the debug signature
# with the key. Then the script checks the signature, that no debug certificate is left, and that the
# alignment is still there after the signing.
#
#   sign-apk.ps1 -Unsigned FILE -Signed FILE -Keystore FILE -BuildTools DIRECTORY
#
# BuildTools is the directory of zipalign and apksigner in the Android SDK. The alias and the passwords
# of the key are in the environment, where the release has them as secrets:
# ANDROID_RELEASE_KEY_ALIAS, ANDROID_RELEASE_STORE_PASSWORD and ANDROID_RELEASE_KEY_PASSWORD.
param(
    [Parameter(Mandatory = $true)][string]$Unsigned,
    [Parameter(Mandatory = $true)][string]$Signed,
    [Parameter(Mandatory = $true)][string]$Keystore,
    [Parameter(Mandatory = $true)][string]$BuildTools
)

$ErrorActionPreference = "Stop"

foreach ($name in "ANDROID_RELEASE_KEY_ALIAS", "ANDROID_RELEASE_STORE_PASSWORD", "ANDROID_RELEASE_KEY_PASSWORD") {
    if (-not [Environment]::GetEnvironmentVariable($name)) { throw "$name is not set" }
}

# The tools of the build tools are .exe and .bat files on Windows, and a binary and a script elsewhere
# (where the script is run to try it).
if ($IsWindows -or $PSVersionTable.PSEdition -eq "Desktop") {
    $zipalign = Join-Path $BuildTools "zipalign.exe"
    $apksigner = Join-Path $BuildTools "apksigner.bat"
} else {
    $zipalign = Join-Path $BuildTools "zipalign"
    $apksigner = Join-Path $BuildTools "apksigner"
}
foreach ($tool in $zipalign, $apksigner, $Unsigned, $Keystore) {
    if (-not (Test-Path -LiteralPath $tool)) { throw "$tool does not exist" }
}

$aligned = Join-Path ([IO.Path]::GetTempPath()) "vfox-plus-aligned-$([guid]::NewGuid()).apk"
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Signed) | Out-Null
try {
    & $zipalign -f -P 16 4 $Unsigned $aligned
    if ($LASTEXITCODE -ne 0) { throw "zipalign failed with exit code $LASTEXITCODE" }

    # The passwords go to apksigner through the environment, not through its command line.
    & $apksigner sign --ks $Keystore --ks-key-alias $env:ANDROID_RELEASE_KEY_ALIAS --ks-pass env:ANDROID_RELEASE_STORE_PASSWORD --key-pass env:ANDROID_RELEASE_KEY_PASSWORD --out $Signed $aligned
    if ($LASTEXITCODE -ne 0) { throw "apksigner sign failed with exit code $LASTEXITCODE" }

    $certificates = & $apksigner verify --print-certs $Signed | Out-String
    if ($LASTEXITCODE -ne 0) { throw "apksigner verify failed with exit code $LASTEXITCODE" }
    if ($certificates -match "CN=Android Debug") { throw "the APK is still signed with the debug key" }

    # Signing keeps the alignment, and the check is what says so.
    & $zipalign -c -P 16 4 $Signed
    if ($LASTEXITCODE -ne 0) { throw "the signed APK is not aligned to 16 KB pages (zipalign -c exit code $LASTEXITCODE)" }
} finally {
    Remove-Item -Force -ErrorAction SilentlyContinue -LiteralPath $aligned
}

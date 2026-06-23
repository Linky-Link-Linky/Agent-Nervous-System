# ANS install script for Windows
# Usage:
#   irm https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.ps1 | iex
# SPDX-License-Identifier: Apache-2.0

$Repo = "Linky-Link-Linky/Agent-Nervous-System"
$Binary = "ans.exe"
$InstallDir = Join-Path $env:USERPROFILE ".ans\bin"
$Version = if ($env:ANS_VERSION) { $env:ANS_VERSION } else { "latest" }

switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default { Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; exit 1 }
}

$Asset = "ans_windows_${Arch}.exe"
$Base = "https://github.com/${Repo}/releases/$($Version)/download"
if ($Version -eq "latest") { $Base = "https://github.com/${Repo}/releases/latest/download" }

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null

try {
    Write-Host "Downloading ANS for Windows/${Arch}..."
    Invoke-WebRequest -Uri "${Base}/${Asset}" -OutFile (Join-Path $TmpDir $Binary)
    Invoke-WebRequest -Uri "${Base}/checksums.txt" -OutFile (Join-Path $TmpDir "checksums.txt")

    $ChecksumLine = Get-Content (Join-Path $TmpDir "checksums.txt") | Select-String -Pattern $Asset
    if (-not $ChecksumLine) {
        Write-Error "Checksum not found for $Asset"; exit 1
    }
    $Expected = ($ChecksumLine -split '\s+')[0]
    $Actual = (Get-FileHash (Join-Path $TmpDir $Binary) -Algorithm SHA256).Hash.ToLower()

    if ($Expected.ToLower() -ne $Actual) {
        Write-Error "Checksum mismatch: expected $Expected, got $Actual"; exit 1
    }

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    Copy-Item (Join-Path $TmpDir $Binary) (Join-Path $InstallDir $Binary) -Force

    $CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($CurrentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
        $env:Path = "$env:Path;$InstallDir"
    }

    Write-Host "ANS installed to $InstallDir"
    Write-Host ""
    Write-Host "Start the daemon:" -ForegroundColor Green
    Write-Host "  ans start" -ForegroundColor Cyan
    Write-Host "Register an agent:"
    Write-Host "  ans register --name my-agent --version 1.0.0"
    Write-Host "View the receipt chain:"
    Write-Host "  ans chain"
    Write-Host ""
    Write-Host "Run 'ans start' now to verify the installation." -ForegroundColor Yellow
}
catch {
    Write-Error "Installation failed: $_"; exit 1
}
finally {
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

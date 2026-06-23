# ANS install script for Windows
# Usage:
#   irm https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.ps1 | iex
# SPDX-License-Identifier: Apache-2.0

# Enable TLS 1.2 for older PowerShell 5.1 (GitHub requires it)
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Repo = "Linky-Link-Linky/Agent-Nervous-System"
$Binary = "ans.exe"
$InstallDir = Join-Path $env:USERPROFILE ".ans\bin"
$Version = if ($env:ANS_VERSION) { $env:ANS_VERSION } else { "latest" }

switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$Asset = "ans_windows_${Arch}.exe"
$Base = "https://github.com/${Repo}/releases/$($Version)/download"
if ($Version -eq "latest") { $Base = "https://github.com/${Repo}/releases/latest/download" }

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null
$script:InstallFailed = $false

try {
    Write-Host "Downloading ANS for Windows/${Arch}..."
    Invoke-WebRequest -Uri "${Base}/${Asset}" -OutFile (Join-Path $TmpDir $Binary)

    # Optional checksum verification — skip if checksums.txt not published yet
    $ChecksumFile = Join-Path $TmpDir "checksums.txt"
    try {
        Invoke-WebRequest -Uri "${Base}/checksums.txt" -OutFile $ChecksumFile -ErrorAction Stop
        $ChecksumLine = Get-Content $ChecksumFile | Select-String -Pattern $Asset
        if ($ChecksumLine) {
            $Expected = ($ChecksumLine -split '\s+')[0].ToLower()
            $Actual = (Get-FileHash (Join-Path $TmpDir $Binary) -Algorithm SHA256).Hash.ToLower()
            if ($Expected -ne $Actual) {
                throw "Checksum mismatch: expected $Expected, got $Actual"
            }
            Write-Host "Checksum verified.`n"
        }
    } catch {
        Write-Host "Checksum file not available — skipping verification." -ForegroundColor Yellow
    }

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }
    Copy-Item (Join-Path $TmpDir $Binary) (Join-Path $InstallDir $Binary) -Force

    # Remove Windows Zone Identifier (downloaded-from-internet marker)
    Unblock-File -Path (Join-Path $InstallDir $Binary) -ErrorAction SilentlyContinue

    # Detect Smart App Control (Windows 11), which blocks unsigned binaries by reputation.
    # When enabled, build-from-source is the only workaround.
    $SacState = (Get-ItemProperty HKLM:\SYSTEM\CurrentControlSet\Control\CI\Policy -Name VerifiedAndReputablePolicyState -ErrorAction SilentlyContinue).VerifiedAndReputablePolicyState
    if ($SacState -eq 1) {
        Write-Host ""
        Write-Host "Smart App Control is ON" -ForegroundColor Yellow
        Write-Host "Windows 11 Smart App Control blocks unsigned binaries downloaded from the internet." -ForegroundColor Yellow
        Write-Host "The downloaded binary will NOT run until you do one of the following:" -ForegroundColor Yellow
        Write-Host ""
        Write-Host "  Option A — Turn off Smart App Control (recommended):" -ForegroundColor Cyan
        Write-Host "    Windows Security > App & browser control > Smart App Control > Off"
        Write-Host ""
        Write-Host "  Option B — Build from source (no restrictions):" -ForegroundColor Cyan
        Write-Host "    cd $(Split-Path $InstallDir)"
        Write-Host "    git clone https://github.com/$Repo.git"
        Write-Host "    cd Agent-Nervous-System\ans"
        Write-Host "    go build -o $(Join-Path $InstallDir $Binary) ./cmd/ans"
        Write-Host ""
        Write-Host "  Option C — Add a path exclusion (may not bypass SAC):" -ForegroundColor Cyan
        Write-Host "    Add-MpPreference -ExclusionPath ""$InstallDir"""
        Write-Host ""
        $script:InstallFailed = $true
    }

    # Only touch User PATH if InstallDir is not already in the effective PATH
    if ($env:Path -split ';' -notcontains $InstallDir) {
        $CurrentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if (-not $CurrentUserPath.EndsWith(';')) { $CurrentUserPath += ';' }
        [Environment]::SetEnvironmentVariable("Path", "${CurrentUserPath}${InstallDir}", "User")
        $env:Path += ";$InstallDir"
    }

    if (-not $script:InstallFailed) {
        Write-Host ""
        Write-Host "ANS installed to $InstallDir" -ForegroundColor Green
        Write-Host ""
        Write-Host "Start the daemon:" -ForegroundColor Cyan
        Write-Host "  ans start"
        Write-Host ""
        Write-Host "Register an agent:"
        Write-Host "  ans register --name my-agent --version 1.0.0"
        Write-Host ""
        Write-Host "View the receipt chain:"
        Write-Host "  ans chain"
        Write-Host ""
        Write-Host "IMPORTANT: Open a NEW PowerShell window before running 'ans'." -ForegroundColor Yellow
        Write-Host "  The PATH change takes effect in new sessions." -ForegroundColor Yellow
    }
}
catch {
    Write-Host ""
    Write-Host "ERROR: Installation failed" -ForegroundColor Red
    Write-Host "  $_" -ForegroundColor Red
    Write-Host ""
    Write-Host "Troubleshooting:" -ForegroundColor Yellow
    Write-Host "  1. Check your internet connection" -ForegroundColor Yellow
    Write-Host "  2. Run: powershell -Command `"`$ProgressPreference='SilentlyContinue'; irm ... | iex`"" -ForegroundColor Yellow
    Write-Host "  3. Or build from source: https://github.com/$Repo" -ForegroundColor Yellow
    Write-Host ""
    throw  # re-throw so the error is visible but PowerShell stays open
}
finally {
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

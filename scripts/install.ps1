# ANS install script for Windows
# Usage: irm https://raw.githubusercontent.com/Linky-Link-Linky/Agent-Nervous-System/master/scripts/install.ps1 | iex
# SPDX-License-Identifier: Apache-2.0

# Enable TLS 1.2 for older PowerShell 5.1
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$Repo = "Linky-Link-Linky/Agent-Nervous-System"
$Binary = "ans.exe"
$InstallDir = Join-Path $env:USERPROFILE ".ans\bin"
$Version = if ($env:ANS_VERSION) { $env:ANS_VERSION } else { "latest" }

# --- Helper functions ---

function Write-Banner {
    Write-Host ""
    Write-Host "  ==========================================" -ForegroundColor Magenta
    Write-Host "       Agent Nervous System" -ForegroundColor Magenta
    Write-Host "       Secure AI Agent Auditing" -ForegroundColor DarkGray
    Write-Host "  ==========================================" -ForegroundColor Magenta
    Write-Host ""
}

function Write-Step($num, $text) {
    Write-Host "  $num. $text" -ForegroundColor Magenta
}

function Write-Done($text) {
    Write-Host "  $([char]0x2714) $text" -ForegroundColor Green
}

function Write-Warn($text) {
    Write-Host "  $([char]0x26A0) $text" -ForegroundColor Yellow
}

function Write-Cmd($text) {
    Write-Host "    $ $text" -ForegroundColor Magenta
}

function Write-Err($text) {
    Write-Host "  $([char]0x2716) $text" -ForegroundColor Red
}

# --- Architecture detection ---

switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default { Write-Err "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"; throw "Unsupported architecture" }
}

$Asset = "ans_windows_${Arch}.exe"
$Base = "https://github.com/${Repo}/releases/download/$($Version)"
if ($Version -eq "latest") { $Base = "https://github.com/${Repo}/releases/latest/download" }

$TmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $TmpDir -Force | Out-Null
$script:BuildFromSource = $false

# --- Main ---

try {
    Clear-Host
    Write-Banner

    # Step 1: Detect platform
    Write-Step 1 "Detecting your system..."
    Write-Host "     Platform: Windows $($Arch)" -ForegroundColor DarkGray
    Write-Host "     Destination: $InstallDir" -ForegroundColor DarkGray
    Start-Sleep -Milliseconds 300

    # Step 2: Check Smart App Control
    Write-Step 2 "Checking Windows security settings..."
    $SacState = (Get-ItemProperty HKLM:\SYSTEM\CurrentControlSet\Control\CI\Policy -Name VerifiedAndReputablePolicyState -ErrorAction SilentlyContinue).VerifiedAndReputablePolicyState
    if ($SacState -eq 1) {
        Write-Warn "Smart App Control is ON"
        Write-Host "     Windows 11 blocks unsigned downloaded binaries by default." -ForegroundColor Yellow
        Write-Host "     I'll build from source instead ? this works everywhere." -ForegroundColor Yellow
        $script:BuildFromSource = $true
    } else {
        Write-Done "Smart App Control is off ? downloading binary"
    }
    Start-Sleep -Milliseconds 300

    if (-not $script:BuildFromSource) {
        # Step 3: Download binary
        Write-Step 3 "Downloading ANS for Windows/${Arch}..."
        Invoke-WebRequest -Uri "${Base}/${Asset}" -OutFile (Join-Path $TmpDir $Binary) -UseBasicParsing
        Write-Done "Downloaded $Asset"

        # Step 4: Optional checksum
        $ChecksumFile = Join-Path $TmpDir "checksums.txt"
        try {
            Invoke-WebRequest -Uri "${Base}/checksums.txt" -OutFile $ChecksumFile -ErrorAction Stop -UseBasicParsing
            $ChecksumLine = Get-Content $ChecksumFile | Select-String -Pattern $Asset
            if ($ChecksumLine) {
                $Expected = ($ChecksumLine -split '\s+')[0].ToLower()
                $Actual = (Get-FileHash (Join-Path $TmpDir $Binary) -Algorithm SHA256).Hash.ToLower()
                if ($Expected -ne $Actual) {
                    throw "Checksum mismatch: expected $Expected, got $Actual"
                }
                Write-Done "Checksum verified"
            }
        } catch {
            Write-Warn "Checksum file not available ? skipped"
        }

        # Step 5: Install binary
        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
        Copy-Item (Join-Path $TmpDir $Binary) (Join-Path $InstallDir $Binary) -Force
        Unblock-File -Path (Join-Path $InstallDir $Binary) -ErrorAction SilentlyContinue
        Write-Done "Binary installed to $InstallDir"
    } else {
        # Build from source (bypasses Smart App Control)
        Write-Step 3 "Building ANS from source..."

        # Check if Go is installed
        $goVer = go version 2>&1
        if ($LASTEXITCODE -ne 0) {
            Write-Warn "Go is not installed. Installing Go first..."
            $goInstaller = Join-Path $TmpDir "go-installer.msi"
            try {
                Invoke-WebRequest -Uri "https://go.dev/dl/go1.25.0.windows-amd64.msi" -OutFile $goInstaller -UseBasicParsing
                Write-Host "     Running Go installer (may show a window)..." -ForegroundColor Yellow
                Start-Process msiexec -ArgumentList "/i `"$goInstaller`" /quiet /norestart" -Wait
                Write-Done "Go installed"
                $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")
            } catch {
                throw "Failed to install Go. Please install it manually from https://go.dev/dl/"
            }
        } else {
            Write-Done "Go is available: $goVer"
        }

        # Clone and build
        $srcDir = Join-Path $env:USERPROFILE ".ans\src"
        if (Test-Path $srcDir) {
            Remove-Item -Path $srcDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        Write-Step 4 "Cloning repository..."
        git clone "https://github.com/$Repo.git" $srcDir 2>&1 | Out-Null
        Write-Done "Repository cloned"

        Write-Step 5 "Building binary..."
        Push-Location $srcDir
        go build -ldflags="-s -w" -trimpath -o (Join-Path $InstallDir $Binary) ./cmd/ans 2>&1
        Pop-Location
        if ($LASTEXITCODE -ne 0) {
            throw "Build failed"
        }
        Write-Done "Binary built and installed"
    }

    # Step 6: Add to PATH
    $StepNum = if ($script:BuildFromSource) { 6 } else { 5 }
Write-Step $StepNum "Adding to system PATH..."
    if ($env:Path -split ';' -notcontains $InstallDir) {
        $CurrentUserPath = [Environment]::GetEnvironmentVariable("Path", "User")
        if (-not $CurrentUserPath.EndsWith(';')) { $CurrentUserPath += ';' }
        [Environment]::SetEnvironmentVariable("Path", "${CurrentUserPath}${InstallDir}", "User")
        $env:Path += ";$InstallDir"
    }
    Write-Done "PATH updated for future terminals"

    # --- Success message ---
    Write-Host ""
    Write-Host "  ==========================================" -ForegroundColor Magenta
    Write-Host "       ANS is installed!" -ForegroundColor Magenta
    Write-Host "  ==========================================" -ForegroundColor Magenta
    Write-Host ""
    Write-Host "  To get started, open a NEW PowerShell window and run:" -ForegroundColor White
    Write-Host ""
    Write-Cmd "ans init"
    Write-Host "      Creates your data directory (~/.ans/) and config" -ForegroundColor DarkGray
    Write-Host ""
    Write-Cmd "ans start"
    Write-Host "      Starts the ANS daemon" -ForegroundColor DarkGray
    Write-Host ""
    Write-Cmd "ans register --name my-agent --version 1.0.0"
    Write-Host "      Register your first AI agent" -ForegroundColor DarkGray
    Write-Host ""
    Write-Cmd "ans chain"
    Write-Host "      View the receipt chain" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "  Need help? Run: ans doctor" -ForegroundColor Magenta
    Write-Host ""
    Write-Host "  Or open a new PowerShell and type: ans init" -ForegroundColor Yellow
    Write-Host ""
}
catch {
    Write-Host ""
    Write-Err "Installation failed: $_"
    Write-Host ""
    Write-Host "  Don't worry! Try one of these:" -ForegroundColor Yellow
    Write-Host "  1. Build from source (works everywhere):" -ForegroundColor Magenta
    Write-Host "     git clone https://github.com/$Repo.git" -ForegroundColor DarkGray
    Write-Host "     cd Agent-Nervous-System/ans" -ForegroundColor DarkGray
    Write-Host "     go build -o ans.exe ./cmd/ans" -ForegroundColor DarkGray
    Write-Host ""
    Write-Host "  2. Get help: https://github.com/$Repo/issues" -ForegroundColor Magenta
    Write-Host ""
    throw
}
finally {
    Remove-Item -Path $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}

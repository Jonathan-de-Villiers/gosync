# GoSync Installation Script for PowerShell
# This script installs GoSync from GitHub releases

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:ProgramFiles\GoSync"
)

# Colors
$Colors = @{
    Red = "Red"
    Green = "Green"
    Yellow = "Yellow"
    Blue = "Blue"
    White = "White"
}

# Functions
function Write-LogInfo {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor $Colors.Blue
}

function Write-LogSuccess {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor $Colors.Green
}

function Write-LogWarning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor $Colors.Yellow
}

function Write-LogError {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor $Colors.Red
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE.ToLower()
    switch ($arch) {
        "amd64" { return "amd64" }
        "x86" { return "386" }
        "arm64" { return "arm64" }
        default { 
            Write-LogError "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# Get latest version
function Get-LatestVersion {
    try {
        $response = Invoke-RestMethod -Uri "https://api.github.com/repos/your-username/gosync/releases/latest" -UseBasicParsing
        return $response.tag_name
    } catch {
        Write-LogError "Failed to get latest version: $_"
        exit 1
    }
}

# Download and install GoSync
function Install-GoSync {
    param([string]$Version, [string]$Arch)
    
    if ($Version -eq "latest") {
        $Version = Get-LatestVersion
    }
    
    $filename = "gosync-$Version-windows-$Arch.zip"
    $downloadUrl = "https://github.com/your-username/gosync/releases/download/$Version/$filename"
    $tempDir = Join-Path $env:TEMP "gosync-install"
    
    Write-LogInfo "Downloading GoSync $Version for Windows-$Arch..."
    
    # Create temp directory
    if (Test-Path $tempDir) {
        Remove-Item -Recurse -Force $tempDir
    }
    New-Item -ItemType Directory -Path $tempDir | Out-Null
    
    # Download
    try {
        Invoke-WebRequest -Uri $downloadUrl -OutFile (Join-Path $tempDir $filename) -UseBasicParsing
    } catch {
        Write-LogError "Failed to download: $_"
        exit 1
    }
    
    # Extract
    Write-LogInfo "Extracting..."
    try {
        Expand-Archive -Path (Join-Path $tempDir $filename) -DestinationPath $tempDir -Force
    } catch {
        Write-LogError "Failed to extract: $_"
        exit 1
    }
    
    # Create install directory
    if (!(Test-Path $InstallDir)) {
        try {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        } catch {
            Write-LogWarning "Cannot create install directory. Using temp directory for installation."
            $InstallDir = $tempDir
        }
    }
    
    # Install binary
    $binaryPath = Join-Path $tempDir "gosync.exe"
    $installPath = Join-Path $InstallDir "gosync.exe"
    
    try {
        Copy-Item $binaryPath $installPath -Force
    } catch {
        Write-LogError "Failed to copy binary: $_"
        exit 1
    }
    
    # Add to PATH if not already there
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        Write-LogInfo "Adding GoSync to PATH..."
        $newPath = $currentPath + ";" + $InstallDir
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        Write-LogWarning "PATH updated. You may need to restart your terminal or run 'refreshenv'."
    }
    
    # Cleanup
    Remove-Item -Recurse -Force $tempDir
    
    Write-LogSuccess "GoSync $Version installed successfully!"
}

# Check if GoSync is already installed
function Test-ExistingInstallation {
    try {
        $result = Get-Command gosync -ErrorAction Stop
        if ($result) {
            try {
                $existingVersion = & gosync --version 2>$null
                Write-LogWarning "GoSync is already installed: $existingVersion"
                $response = Read-Host "Do you want to continue and overwrite? (y/N)"
                if ($response -notmatch '^[Yy]') {
                    Write-LogInfo "Installation cancelled"
                    exit 0
                }
            } catch {
                # Command exists but doesn't work properly
            }
        }
    } catch {
        # GoSync not found, proceed with installation
    }
}

# Main installation
function Main {
    Write-LogInfo "GoSync Installation Script for PowerShell"
    Write-LogInfo "Repository: your-username/gosync"
    Write-LogInfo "Version: $Version"
    Write-LogInfo "Install Directory: $InstallDir"
    
    # Check if running as administrator for system-wide installation
    $isAdmin = ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    
    if (!$isAdmin -and $InstallDir -like "*Program Files*") {
        Write-LogWarning "Installing to Program Files requires administrator privileges."
        Write-LogInfo "You can specify a different directory: .\scripts\install.ps1 -InstallDir 'C:\GoSync'"
        $response = Read-Host "Continue with administrator installation? (y/N)"
        if ($response -notmatch '^[Yy]') {
            Write-LogInfo "Installation cancelled"
            exit 0
        }
    }
    
    # Check existing installation
    Test-ExistingInstallation
    
    # Detect architecture
    $arch = Get-Architecture
    Write-LogInfo "Detected architecture: Windows-$arch"
    
    # Install
    Install-GoSync -Version $Version -Arch $arch
    
    # Verify installation
    try {
        $result = & gosync --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-LogSuccess "Installation verified! Run 'gosync --help' to get started."
        } else {
            Write-LogError "Installation verification failed"
            exit 1
        }
    } catch {
        Write-LogError "Installation verification failed: $_"
        Write-LogInfo "You may need to restart your terminal or run 'refreshenv' to use the updated PATH."
    }
}

# Run main function
Main

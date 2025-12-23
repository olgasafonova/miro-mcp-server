# Miro MCP Server Installer for Windows
# Run as: powershell -ExecutionPolicy Bypass -File install.ps1

$ErrorActionPreference = "Stop"

Write-Host "Installing Miro MCP Server..." -ForegroundColor Cyan

# Get latest release version
try {
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/olgasafonova/miro-mcp-server/releases/latest"
    $version = $release.tag_name
    Write-Host "Latest version: $version"
} catch {
    Write-Host "Failed to get latest version. Using 'latest'." -ForegroundColor Yellow
    $version = "latest"
}

# Set download URL and destination
$downloadUrl = "https://github.com/olgasafonova/miro-mcp-server/releases/download/$version/miro-mcp-server-windows-amd64.exe"
$installDir = "$env:LOCALAPPDATA\Programs\miro-mcp-server"
$exePath = "$installDir\miro-mcp-server.exe"

# Create install directory
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Download binary
Write-Host "Downloading from $downloadUrl..."
try {
    Invoke-WebRequest -Uri $downloadUrl -OutFile $exePath -UseBasicParsing
} catch {
    Write-Host "Download failed: $_" -ForegroundColor Red
    exit 1
}

# Verify download
if (-not (Test-Path $exePath)) {
    Write-Host "Installation failed - file not found" -ForegroundColor Red
    exit 1
}

# Add to PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    Write-Host "Adding to PATH..."
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    $env:Path = "$env:Path;$installDir"
}

Write-Host ""
Write-Host "Installation complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Installed to: $exePath"
Write-Host "Version: $version"
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Get a Miro token: https://miro.com/app/settings/user-profile/apps"
Write-Host "2. Set environment variable: `$env:MIRO_ACCESS_TOKEN = 'your-token'"
Write-Host "3. Configure your AI tool (see SETUP.md)"
Write-Host ""
Write-Host "Restart your terminal for PATH changes to take effect."

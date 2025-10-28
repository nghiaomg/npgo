Write-Host "🚀 Installing NPGO..." -ForegroundColor Cyan

$ErrorActionPreference = 'Stop'

try {
    $Repo = 'nghiaomg/npgo'
    $Url  = "https://github.com/$Repo/releases/latest/download/npgo.exe"
    $InstallPath = Join-Path $Env:ProgramFiles 'npgo'

    # Create install directory
    New-Item -ItemType Directory -Force -Path $InstallPath | Out-Null

    Write-Host "⬇️  Downloading from $Url" -ForegroundColor DarkCyan
    $Dest = Join-Path $InstallPath 'npgo.exe'
    Invoke-WebRequest -Uri $Url -OutFile $Dest -UseBasicParsing

    # Make sure it's executable
    if (-not (Test-Path $Dest)) {
        throw "Download failed: $Dest not found"
    }

    # Add to PATH (Machine if possible, fallback to User)
    $pathToAdd = $InstallPath
    $machineUpdated = $false
    try {
        $machinePath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
        if ($machinePath -notlike "*${pathToAdd}*") {
            [Environment]::SetEnvironmentVariable('Path', "$machinePath;${pathToAdd}", 'Machine')
        }
        $machineUpdated = $true
    } catch {
        # ignore; will fallback to User scope
    }

    if (-not $machineUpdated) {
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        if ($userPath -notlike "*${pathToAdd}*") {
            [Environment]::SetEnvironmentVariable('Path', "$userPath;${pathToAdd}", 'User')
        }
    }

    # Update current session PATH
    if ($Env:Path -notlike "*${pathToAdd}*") {
        $Env:Path = "$Env:Path;${pathToAdd}"
    }

    Write-Host "✅ Installed NPGO to $InstallPath" -ForegroundColor Green
    Write-Host "ℹ️  You may need to open a new terminal for PATH changes to take effect." -ForegroundColor Yellow

    # Verify
    Write-Host ""; Write-Host "Running 'npgo' to verify:" -ForegroundColor DarkCyan
    & npgo --help
}
catch {
    Write-Host "❌ Installation failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}



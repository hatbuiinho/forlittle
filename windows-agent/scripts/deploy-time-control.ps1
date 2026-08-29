[CmdletBinding()]
param(
    [string]$ReleaseDirectory,
    [switch]$ForceReenroll
)

$ErrorActionPreference = "Stop"

function Test-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Parameter defaults are evaluated before $PSScriptRoot is populated on some
# Windows PowerShell hosts. Resolve the release directory only after loading.
if ([string]::IsNullOrWhiteSpace($ReleaseDirectory)) {
    $ReleaseDirectory = $PSScriptRoot
}

if (-not (Test-Administrator)) {
    $arguments = @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", "`"$PSCommandPath`"",
        "-ReleaseDirectory", "`"$ReleaseDirectory`""
    )
    if ($ForceReenroll) {
        $arguments += "-ForceReenroll"
    }
    $process = Start-Process -FilePath "powershell.exe" -Verb RunAs -Wait -PassThru -ArgumentList $arguments
    exit $process.ExitCode
}

$errorLog = Join-Path $PSScriptRoot "deploy-error.log"

try {
    $ReleaseDirectory = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
    $errorLog = Join-Path $ReleaseDirectory "deploy-error.log"
    $serviceExecutable = Join-Path $ReleaseDirectory "forlittle-time-control.exe"
    $agentDirectory = Join-Path $ReleaseDirectory "agent"
    $agentExecutable = Join-Path $agentDirectory "ForLittle.TimeControl.Agent.exe"
    $configPath = Join-Path $ReleaseDirectory "config.json"
    $configTemplate = Join-Path $ReleaseDirectory "config.time-control.example.json"
    $installer = Join-Path $ReleaseDirectory "install-time-control.ps1"
    $serviceName = "ForLittleTimeControl"
    $dataDirectory = "C:\ProgramData\ForLittle\TimeControl"

    foreach ($path in @($serviceExecutable, $agentExecutable, $installer)) {
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw "Missing required release file: $path"
        }
    }

    if (-not (Test-Path -LiteralPath $configPath -PathType Leaf)) {
        if (-not (Test-Path -LiteralPath $configTemplate -PathType Leaf)) {
            throw "Missing config.json and config.time-control.example.json in $ReleaseDirectory"
        }
        Copy-Item -LiteralPath $configTemplate -Destination $configPath
        Start-Process -FilePath "notepad.exe" -ArgumentList "`"$configPath`""
        throw "Created $configPath. Fill in server_url, machine_id, little_monk_code, little_monk_display_name, and enrollment_key, save it, then run this script again."
    }

    $config = Get-Content -LiteralPath $configPath -Raw | ConvertFrom-Json
    foreach ($name in @("server_url", "machine_id", "little_monk_code", "enrollment_key")) {
        if ([string]::IsNullOrWhiteSpace([string]$config.$name)) {
            throw "config.json is missing $name"
        }
    }
    if ($config.server_url -notmatch "^https?://") {
        throw "server_url must start with http:// or https://"
    }

    if ($ForceReenroll) {
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        Remove-Item -LiteralPath (Join-Path $dataDirectory "credentials.json") -Force -ErrorAction SilentlyContinue
    }

    & $installer `
        -ServiceExecutable $serviceExecutable `
        -AgentExecutable $agentExecutable `
        -AgentDirectory $agentDirectory `
        -ConfigPath $configPath

    $service = Get-Service -Name $serviceName
    if ($service.Status -ne "Running") {
        throw "Service $serviceName did not start. Inspect $dataDirectory\service.log"
    }

    Write-Host "For Little Time Control is running." -ForegroundColor Green
    Write-Host "Log: $dataDirectory\service.log"
    Write-Host "The dashboard will show $($config.little_monk_display_name) after successful enrollment."
}
catch {
    $details = "$(Get-Date -Format o)`r`n$($_ | Out-String)"
    Set-Content -LiteralPath $errorLog -Value $details -Encoding utf8
    Write-Host "Deployment failed: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "Details: $errorLog" -ForegroundColor Yellow
    Read-Host "Press Enter to close"
    exit 1
}

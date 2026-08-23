[CmdletBinding()]
param(
    [string]$ReleaseDirectory = $PSScriptRoot,
    [switch]$ForceReenroll
)

$ErrorActionPreference = "Stop"

function Test-Administrator {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = [Security.Principal.WindowsPrincipal]::new($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
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

$ReleaseDirectory = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
$serviceExecutable = Join-Path $ReleaseDirectory "forlittle-time-control.exe"
$agentExecutable = Join-Path $ReleaseDirectory "ForLittle.TimeControl.Agent.exe"
$configPath = Join-Path $ReleaseDirectory "config.json"
$installer = Join-Path $ReleaseDirectory "install-time-control.ps1"
$serviceName = "ForLittleTimeControl"
$dataDirectory = "C:\ProgramData\ForLittle\TimeControl"

foreach ($path in @($serviceExecutable, $agentExecutable, $configPath, $installer)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Missing required release file: $path"
    }
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
    -ConfigPath $configPath

$service = Get-Service -Name $serviceName
if ($service.Status -ne "Running") {
    throw "Service $serviceName did not start. Inspect $dataDirectory\service.log"
}

Write-Host "For Little Time Control is running." -ForegroundColor Green
Write-Host "Log: $dataDirectory\service.log"
Write-Host "The dashboard will show $($config.little_monk_display_name) after successful enrollment."

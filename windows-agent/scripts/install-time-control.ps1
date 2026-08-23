[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ServiceExecutable,

    [Parameter(Mandatory = $true)]
    [string]$AgentExecutable,

    [Parameter(Mandatory = $true)]
    [string]$ConfigPath,

    [string]$InstallDirectory = "C:\Program Files\ForLittle\TimeControl",
    [string]$DataDirectory = "C:\ProgramData\ForLittle\TimeControl",
    [string]$ServiceName = "ForLittleTimeControl"
)

$ErrorActionPreference = "Stop"

foreach ($path in @($ServiceExecutable, $AgentExecutable, $ConfigPath)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required file was not found: $path"
    }
}

New-Item -ItemType Directory -Path $InstallDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $DataDirectory -Force | Out-Null

$installedService = Join-Path $InstallDirectory "forlittle-time-control.exe"
$installedAgent = Join-Path $InstallDirectory "ForLittle.TimeControl.Agent.exe"
$installedConfig = Join-Path $DataDirectory "config.json"

if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
}

Copy-Item -LiteralPath $ServiceExecutable -Destination $installedService -Force
Copy-Item -LiteralPath $AgentExecutable -Destination $installedAgent -Force
Copy-Item -LiteralPath $ConfigPath -Destination $installedConfig -Force

$serviceConfig = Get-Content -LiteralPath $installedConfig -Raw | ConvertFrom-Json
$serviceConfig.agent_path = $installedAgent
$serviceConfig.data_dir = $DataDirectory
$serviceConfig | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $installedConfig -Encoding utf8

# Program Files is protected by Windows. Restrict mutable service state so a
# Standard User cannot replace credentials, policy cache, or command history.
& icacls.exe $DataDirectory /inheritance:r | Out-Null
& icacls.exe $DataDirectory /grant:r "SYSTEM:(OI)(CI)F" "Administrators:(OI)(CI)F" | Out-Null

if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
    & sc.exe create $ServiceName binPath= "`"$installedService`" -config `"$installedConfig`"" start= delayed-auto DisplayName= "For Little Time Control" | Out-Null
}
else {
    & sc.exe config $ServiceName binPath= "`"$installedService`" -config `"$installedConfig`"" start= delayed-auto | Out-Null
}

& sc.exe failure $ServiceName reset= 86400 actions= restart/5000/restart/5000/restart/5000 | Out-Null
& sc.exe failureflag $ServiceName 1 | Out-Null
Start-Service -Name $ServiceName

Write-Host "Installed $ServiceName. Verify with: Get-Service $ServiceName"

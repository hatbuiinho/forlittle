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

function Invoke-Sc {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Arguments)

    & "$env:SystemRoot\System32\sc.exe" @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "sc.exe $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

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
$configJSON = $serviceConfig | ConvertTo-Json -Depth 8
[System.IO.File]::WriteAllText($installedConfig, $configJSON, [System.Text.UTF8Encoding]::new($false))

# Program Files is protected by Windows. Restrict mutable service state so a
# Standard User cannot replace credentials, policy cache, or command history.
& icacls.exe $DataDirectory /inheritance:r | Out-Null
& icacls.exe $DataDirectory /grant:r "SYSTEM:(OI)(CI)F" "Administrators:(OI)(CI)F" | Out-Null

$serviceCommandLine = "`"$installedService`" -config `"$installedConfig`""
if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
    New-Service `
        -Name $ServiceName `
        -BinaryPathName $serviceCommandLine `
        -DisplayName "For Little Time Control" `
        -StartupType Automatic | Out-Null
}
else {
    Invoke-Sc config $ServiceName binPath= $serviceCommandLine
}

Invoke-Sc config $ServiceName start= delayed-auto
Invoke-Sc failure $ServiceName reset= 86400 actions= restart/5000/restart/5000/restart/5000
Invoke-Sc failureflag $ServiceName 1
Start-Service -Name $ServiceName

Write-Host "Installed $ServiceName. Verify with: Get-Service $ServiceName"

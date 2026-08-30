[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ServiceExecutable,

    [Parameter(Mandatory = $true)]
    [string]$AgentExecutable,

    [Parameter(Mandatory = $true)]
    [string]$AgentDirectory,

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

function Set-ServiceCommandLine {
    param(
        [string]$Name,
        [string]$CommandLine
    )

    $service = Get-CimInstance -ClassName Win32_Service -Filter "Name='$Name'"
    if ($null -eq $service) {
        throw "Service $Name was not found"
    }
    $result = Invoke-CimMethod -InputObject $service -MethodName Change -Arguments @{
        PathName = $CommandLine
        StartMode = "Automatic"
        DisplayName = "For Little Time Control"
    }
    if ($result.ReturnValue -ne 0) {
        throw "Could not update service $Name (Win32_Service.Change returned $($result.ReturnValue))"
    }
}

foreach ($path in @($ServiceExecutable, $AgentExecutable, $ConfigPath)) {
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Required file was not found: $path"
    }
}
if (-not (Test-Path -LiteralPath $AgentDirectory -PathType Container)) {
    throw "Agent directory was not found: $AgentDirectory"
}

New-Item -ItemType Directory -Path $InstallDirectory -Force | Out-Null
New-Item -ItemType Directory -Path $DataDirectory -Force | Out-Null

$installedService = Join-Path $InstallDirectory "forlittle-time-control.exe"
$installedAgentDirectory = Join-Path $InstallDirectory "agent"
$installedAgent = Join-Path $installedAgentDirectory "ForLittle.TimeControl.Agent.exe"
$legacyAgent = Join-Path $InstallDirectory "ForLittle.TimeControl.Agent.exe"
$startMenuDirectory = Join-Path ([Environment]::GetFolderPath("CommonPrograms")) "For Little"
$scheduleShortcut = Join-Path $startMenuDirectory "Lich dung may cua Chu Tieu.lnk"
$installedConfig = Join-Path $DataDirectory "config.json"

if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
}

Copy-Item -LiteralPath $ServiceExecutable -Destination $installedService -Force
Remove-Item -LiteralPath $installedAgentDirectory -Recurse -Force -ErrorAction SilentlyContinue
Copy-Item -LiteralPath $AgentDirectory -Destination $installedAgentDirectory -Recurse -Force
# Older releases placed the agent directly in the install directory. Remove it
# after the service is stopped so manual starts cannot launch a stale runtime.
Remove-Item -LiteralPath $legacyAgent -Force -ErrorAction SilentlyContinue
Copy-Item -LiteralPath $ConfigPath -Destination $installedConfig -Force

$serviceConfig = Get-Content -LiteralPath $installedConfig -Raw | ConvertFrom-Json
$serviceConfig.agent_path = $installedAgent
$serviceConfig.data_dir = $DataDirectory
$configJSON = $serviceConfig | ConvertTo-Json -Depth 8
[System.IO.File]::WriteAllText($installedConfig, $configJSON, [System.Text.UTF8Encoding]::new($false))

New-Item -ItemType Directory -Path $startMenuDirectory -Force | Out-Null
$shell = New-Object -ComObject WScript.Shell
$shortcut = $shell.CreateShortcut($scheduleShortcut)
$shortcut.TargetPath = $installedAgent
$shortcut.Arguments = "--show-schedule"
$shortcut.WorkingDirectory = $installedAgentDirectory
$shortcut.Description = "Xem lich dung may da duoc ap dung"
$shortcut.Save()

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
    Set-ServiceCommandLine -Name $ServiceName -CommandLine $serviceCommandLine
}

# Start immediately at boot so the active Standard User receives time-control
# state without waiting for Windows delayed auto-start.
Invoke-Sc config $ServiceName start= auto
Invoke-Sc failure $ServiceName reset= 86400 actions= restart/5000/restart/5000/restart/5000
Invoke-Sc failureflag $ServiceName 1
Start-Service -Name $ServiceName

Write-Host "Installed $ServiceName. Verify with: Get-Service $ServiceName"

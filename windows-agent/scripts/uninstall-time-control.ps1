[CmdletBinding()]
param(
    [string]$InstallDirectory = "C:\Program Files\ForLittle\TimeControl",
    [string]$DataDirectory = "C:\ProgramData\ForLittle\TimeControl",
    [string]$ServiceName = "ForLittleTimeControl"
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
        "-InstallDirectory", "`"$InstallDirectory`"",
        "-DataDirectory", "`"$DataDirectory`"",
        "-ServiceName", "`"$ServiceName`""
    )
    $process = Start-Process -FilePath "powershell.exe" -Verb RunAs -Wait -PassThru -ArgumentList $arguments
    exit $process.ExitCode
}

try {
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue

    # The agent runs in the interactive user session, outside the service tree.
    Get-Process -Name "ForLittle.TimeControl.Agent" -ErrorAction SilentlyContinue |
        Stop-Process -Force -ErrorAction SilentlyContinue

    if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
        & "$env:SystemRoot\System32\sc.exe" delete $ServiceName | Out-Null
        if ($LASTEXITCODE -ne 0) {
            throw "Could not delete Windows Service $ServiceName (sc.exe exit code $LASTEXITCODE)"
        }
    }

    Remove-Item -LiteralPath $InstallDirectory -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $DataDirectory -Recurse -Force -ErrorAction SilentlyContinue

    $scheduleShortcut = Join-Path ([Environment]::GetFolderPath("CommonPrograms")) "For Little\Lich dung may cua Chu Tieu.lnk"
    Remove-Item -LiteralPath $scheduleShortcut -Force -ErrorAction SilentlyContinue

    $userAgentData = Join-Path $env:LOCALAPPDATA "ForLittle\TimeControl"
    Remove-Item -LiteralPath $userAgentData -Recurse -Force -ErrorAction SilentlyContinue

    Write-Host "For Little Time Control was removed." -ForegroundColor Green
    Write-Host "The dashboard machine record was retained. Reinstalling with the same machine_id will enroll it again."
}
catch {
    Write-Host "Uninstall failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}

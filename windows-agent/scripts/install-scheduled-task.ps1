param(
  [Parameter(Mandatory = $true)]
  [string]$AgentPath,

  [Parameter(Mandatory = $true)]
  [string]$ConfigPath,

  [string]$TaskName = "ForLittleAgent"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $AgentPath)) {
  throw "Agent executable not found: $AgentPath"
}

if (-not (Test-Path $ConfigPath)) {
  throw "Config file not found: $ConfigPath"
}

$action = New-ScheduledTaskAction -Execute $AgentPath -Argument "-config `"$ConfigPath`""
$trigger = New-ScheduledTaskTrigger -AtLogOn
$principal = New-ScheduledTaskPrincipal -GroupId "Users" -RunLevel Highest
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -RestartCount 999 -RestartInterval (New-TimeSpan -Minutes 1)

Register-ScheduledTask -TaskName $TaskName -Action $action -Trigger $trigger -Principal $principal -Settings $settings -Force

Write-Host "Installed scheduled task: $TaskName"

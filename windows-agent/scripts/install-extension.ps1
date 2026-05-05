param(
  [Parameter(Mandatory = $true)]
  [string]$SourceExtensionPath,

  [string]$TargetExtensionPath = "C:\ProgramData\ForLittle\Extension",
  [string]$TargetProfilePath = "C:\ProgramData\ForLittle\ChromeUserData"
)

$ErrorActionPreference = "Stop"

$manifestPath = Join-Path $SourceExtensionPath "manifest.json"
if (-not (Test-Path $manifestPath)) {
  throw "SourceExtensionPath must point to the unpacked extension folder containing manifest.json: $SourceExtensionPath"
}

New-Item -ItemType Directory -Force -Path $TargetExtensionPath | Out-Null
New-Item -ItemType Directory -Force -Path $TargetProfilePath | Out-Null

Get-ChildItem -Path $TargetExtensionPath -Force | Remove-Item -Recurse -Force
Copy-Item -Path (Join-Path $SourceExtensionPath "*") -Destination $TargetExtensionPath -Recurse -Force

icacls $TargetExtensionPath /inheritance:e | Out-Null
icacls $TargetExtensionPath /grant "Users:(OI)(CI)RX" /T | Out-Null

icacls $TargetProfilePath /inheritance:e | Out-Null
icacls $TargetProfilePath /grant "Users:(OI)(CI)M" /T | Out-Null

Write-Host "Installed extension to: $TargetExtensionPath"
Write-Host "Prepared Chrome profile path: $TargetProfilePath"
Write-Host "Use this extension_path in config.json: $TargetExtensionPath"
Write-Host "Use this profile_path in config.json: $TargetProfilePath"

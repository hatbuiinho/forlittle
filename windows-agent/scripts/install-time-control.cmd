@echo off
setlocal

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0deploy-time-control.ps1"
set "EXIT_CODE=%ERRORLEVEL%"

if not "%EXIT_CODE%"=="0" (
  echo.
  echo Cai dat Time Control that bai. Xem file deploy-error.log trong thu muc release.
  pause
)

exit /b %EXIT_CODE%

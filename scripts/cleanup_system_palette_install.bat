@echo off
setlocal EnableExtensions

rem Remove old all-users/admin Palette install artifacts so the per-user
rem installer is the only active Palette install.

if /i "%~1"=="--elevated" shift & goto elevated

net session >nul 2>&1
if errorlevel 1 (
    echo Requesting Administrator privileges to clean up the old all-users Palette install...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Start-Process powershell -Verb RunAs -ArgumentList '-NoProfile -ExecutionPolicy Bypass -Command ""& ''%~f0'' --elevated %*""'"
    exit /b
)

:elevated
set "DELETE_DATA=0"
set "QUIET=0"

:parse_args
if "%~1"=="" goto args_done
if /i "%~1"=="/delete-data" set "DELETE_DATA=1"
if /i "%~1"=="--delete-data" set "DELETE_DATA=1"
if /i "%~1"=="/quiet" set "QUIET=1"
if /i "%~1"=="--quiet" set "QUIET=1"
shift
goto parse_args

:args_done
echo Cleaning old all-users Palette install...
echo.

echo Stopping Palette processes...
taskkill /F /T /IM palette_engine.exe >nul 2>nul
taskkill /F /T /IM palette_monitor.exe >nul 2>nul
powershell -NoProfile -ExecutionPolicy Bypass -Command "Get-CimInstance Win32_Process | Where-Object { $_.CommandLine -and $_.CommandLine -match 'samplesplitter\.py' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }" >nul 2>nul

set "PROGRAM_INSTALL=%ProgramFiles%\Palette"
set "SYSTEM_DATA=%ProgramFiles%\Common"
set "SYSTEM_DATA=%SYSTEM_DATA% Files\Palette"
set "USER_DATA=%LOCALAPPDATA%\Palette"

if exist "%SYSTEM_DATA%" (
    echo Copying old system-wide logs into the per-user data area...
    if not exist "%USER_DATA%\logs\system_install" mkdir "%USER_DATA%\logs\system_install" >nul 2>nul
    if exist "%SYSTEM_DATA%\logs" (
        robocopy "%SYSTEM_DATA%\logs" "%USER_DATA%\logs\system_install" /E /XO >nul
        if errorlevel 8 echo Warning: failed to copy logs from "%SYSTEM_DATA%\logs"
    )
    for /d %%D in ("%SYSTEM_DATA%\data_*") do (
        if exist "%%~fD\logs" (
            if not exist "%USER_DATA%\%%~nxD\logs" mkdir "%USER_DATA%\%%~nxD\logs" >nul 2>nul
            robocopy "%%~fD\logs" "%USER_DATA%\%%~nxD\logs" /E /XO >nul
            if errorlevel 8 echo Warning: failed to copy logs from "%%~fD\logs"
        )
    )
)

if exist "%PROGRAM_INSTALL%\unins000.exe" (
    echo Running old all-users app uninstaller...
    "%PROGRAM_INSTALL%\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
)

if exist "%SYSTEM_DATA%\unins000.exe" (
    echo Running old all-users data uninstaller...
    "%SYSTEM_DATA%\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
)

echo Removing machine-wide environment variables and PATH entries...
powershell -NoProfile -ExecutionPolicy Bypass -Command ^
  "$envKey='HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment';" ^
  "foreach($name in 'PALETTE','PALETTE_DATA','PALETTE_DATAROOT'){ Remove-ItemProperty -Path $envKey -Name $name -ErrorAction SilentlyContinue };" ^
  "$path=(Get-ItemProperty -Path $envKey -Name Path -ErrorAction SilentlyContinue).Path;" ^
  "if($path){ $remove=@('%ProgramFiles%\Palette\bin','%ProgramFiles%\Palette\ffgl'); $parts=$path -split ';' | Where-Object { $p=$_.Trim(); $p -and ($remove -notcontains $p) }; Set-ItemProperty -Path $envKey -Name Path -Value (($parts | Select-Object -Unique) -join ';') }"

if exist "%PROGRAM_INSTALL%" (
    echo Removing old all-users app directory: %PROGRAM_INSTALL%
    rmdir /S /Q "%PROGRAM_INSTALL%" >nul 2>nul
)

if exist "%SYSTEM_DATA%" (
    if "%DELETE_DATA%"=="1" (
        echo Deleting old all-users data directory: %SYSTEM_DATA%
        powershell -NoProfile -ExecutionPolicy Bypass -Command ^
          "$ErrorActionPreference='Stop';" ^
          "Remove-Item -LiteralPath $env:SYSTEM_DATA -Recurse -Force"
    ) else (
        echo Moving old all-users data directory to a timestamped backup...
        powershell -NoProfile -ExecutionPolicy Bypass -Command ^
          "$ErrorActionPreference='Stop';" ^
          "$backup=Join-Path (Split-Path -Parent $env:SYSTEM_DATA) ('Palette_system_backup_' + (Get-Date -Format 'yyyyMMdd_HHmmss'));" ^
          "Move-Item -LiteralPath $env:SYSTEM_DATA -Destination $backup -Force;" ^
          "Write-Host ('    ' + $backup)"
    )
    if errorlevel 1 (
        echo ERROR: failed to remove or move old all-users data directory:
        echo     %SYSTEM_DATA%
        echo Re-run this script as Administrator, or move that directory manually.
        goto failed
    )
)

if exist "%SYSTEM_DATA%" (
    echo ERROR: old all-users data directory still exists:
    echo     %SYSTEM_DATA%
    echo The per-user installer will keep refusing to install until this path is gone.
    goto failed
)

echo.
echo Done. Open a new terminal so user-level PALETTE environment changes are visible.
echo.
if not "%QUIET%"=="1" pause
exit /b 0

:failed
echo.
if not "%QUIET%"=="1" pause
exit /b 1

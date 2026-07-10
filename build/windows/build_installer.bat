@echo off
setlocal EnableExtensions

if "%~1"=="" goto usage
if "%~2"=="" goto usage
if "%~3"=="" goto usage
if "%~4"=="" goto usage
if "%PALETTE_SOURCE%"=="" (
	echo PALETTE_SOURCE must be set.
	exit /b 1
)

set "INSTALLER_KIND=%~1"
set "INSTALLER_SOURCE=%~2"
set "INSTALLER_OUTPUT=%~3"
set "INSTALLER_VERSION=%~4"
set "INSTALLER_DATA_NAME=%~5"
set "INSTALLER_DELETE=%~6"
set "INSTALLER_STUB=%TEMP%\palette_installer_stub_%RANDOM%_%RANDOM%.exe"

echo Building bespoke Palette installer stub...
go build -ldflags "-H=windowsgui" -o "%INSTALLER_STUB%" "%PALETTE_SOURCE%\cmd\palette_installer"
if errorlevel 1 goto failed

echo Packaging %INSTALLER_KIND% installer...
if "%INSTALLER_DELETE%"=="" goto package_without_delete
go run "%PALETTE_SOURCE%\cmd\palette_installer_packager" -stub "%INSTALLER_STUB%" -source "%INSTALLER_SOURCE%" -output "%INSTALLER_OUTPUT%" -kind "%INSTALLER_KIND%" -version "%INSTALLER_VERSION%" -data-name "%INSTALLER_DATA_NAME%" -delete "%INSTALLER_DELETE%"
goto package_done

:package_without_delete
go run "%PALETTE_SOURCE%\cmd\palette_installer_packager" -stub "%INSTALLER_STUB%" -source "%INSTALLER_SOURCE%" -output "%INSTALLER_OUTPUT%" -kind "%INSTALLER_KIND%" -version "%INSTALLER_VERSION%" -data-name "%INSTALLER_DATA_NAME%"

:package_done
if errorlevel 1 goto failed
del /q "%INSTALLER_STUB%" >nul 2>&1
exit /b 0

:usage
echo Usage: build_installer kind source output version [data-name] [delete-list]
exit /b 2

:failed
del /q "%INSTALLER_STUB%" >nul 2>&1
echo Failed to create %INSTALLER_KIND% installer.
exit /b 1

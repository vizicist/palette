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
set "INSTALLER_PACKER=%TEMP%\palette_packer_%RANDOM%_%RANDOM%.exe"

call :find_manifest_tool
if errorlevel 1 goto failed

echo Building bespoke Palette installer stub...
go build -ldflags "-H=windowsgui" -o "%INSTALLER_STUB%" "%PALETTE_SOURCE%\cmd\palette_installer"
if errorlevel 1 goto failed

rem An explicit asInvoker manifest prevents Windows installer detection from
rem demanding elevation for this per-user executable, including 32-bit builds.
"%MT_EXE%" -nologo -manifest "%PALETTE_SOURCE%\cmd\palette_installer\app.manifest" "-outputresource:%INSTALLER_STUB%;#1"
if errorlevel 1 goto failed

rem Do not let go run name its temporary executable palette_installer_packager:
rem Windows can treat executable names containing "install" as elevation-worthy.
go build -o "%INSTALLER_PACKER%" "%PALETTE_SOURCE%\cmd\palette_installer_packager"
if errorlevel 1 goto failed
"%MT_EXE%" -nologo -manifest "%PALETTE_SOURCE%\cmd\palette_installer\app.manifest" "-outputresource:%INSTALLER_PACKER%;#1"
if errorlevel 1 goto failed

echo Packaging %INSTALLER_KIND% installer...
if "%INSTALLER_DELETE%"=="" goto package_without_delete
"%INSTALLER_PACKER%" -stub "%INSTALLER_STUB%" -source "%INSTALLER_SOURCE%" -output "%INSTALLER_OUTPUT%" -kind "%INSTALLER_KIND%" -version "%INSTALLER_VERSION%" -data-name "%INSTALLER_DATA_NAME%" -delete "%INSTALLER_DELETE%"
goto package_done

:package_without_delete
"%INSTALLER_PACKER%" -stub "%INSTALLER_STUB%" -source "%INSTALLER_SOURCE%" -output "%INSTALLER_OUTPUT%" -kind "%INSTALLER_KIND%" -version "%INSTALLER_VERSION%" -data-name "%INSTALLER_DATA_NAME%"

:package_done
if errorlevel 1 goto failed
del /q "%INSTALLER_STUB%" >nul 2>&1
del /q "%INSTALLER_PACKER%" >nul 2>&1
exit /b 0

:usage
echo Usage: build_installer kind source output version [data-name] [delete-list]
exit /b 2

:failed
del /q "%INSTALLER_STUB%" >nul 2>&1
del /q "%INSTALLER_PACKER%" >nul 2>&1
echo Failed to create %INSTALLER_KIND% installer.
exit /b 1

:find_manifest_tool
set "MT_EXE="
for %%i in (mt.exe) do if not "%%~$PATH:i"=="" set "MT_EXE=%%~$PATH:i"
if defined MT_EXE exit /b 0
if defined WindowsSdkVerBinPath if exist "%WindowsSdkVerBinPath%x64\mt.exe" set "MT_EXE=%WindowsSdkVerBinPath%x64\mt.exe"
if defined MT_EXE exit /b 0
for /d %%i in ("%ProgramFiles(x86)%\Windows Kits\10\bin\*") do if exist "%%~fi\x64\mt.exe" set "MT_EXE=%%~fi\x64\mt.exe"
if defined MT_EXE exit /b 0
echo The Windows SDK Manifest Tool, mt.exe, was not found.
echo Install the Windows SDK with Visual Studio C++ Build Tools.
exit /b 1

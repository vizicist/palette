@echo off

set /p version=<../../VERSION

set data=%1

if not "%data%" == "" goto keepgoing0
	echo You must provide an argument, e.g. "default"
	goto getout
:keepgoing0

if not "%PALETTE_SOURCE%" == "" goto keepgoing1
	echo You must set the PALETTE_SOURCE environment variable.
	goto getout
:keepgoing1
for %%I in ("%PALETTE_SOURCE%") do set "PALETTE_SOURCE=%%~fI"

rem Refuse to run against anything that is not a Palette source tree.
rem Every destructive path below is built from these variables, so an empty
rem or mistaken PALETTE_SOURCE aims them somewhere else entirely.
if not exist "%PALETTE_SOURCE%\VERSION" (
	echo ERROR: PALETTE_SOURCE does not look like a Palette source tree:
	echo     %PALETTE_SOURCE%
	exit /b 1
)

set ship=%PALETTE_SOURCE%\build\windows\ship
set datadir=data_%data%

rm -fr "%ship%" > nul 2>&1
mkdir %ship%

echo ================ Copying %datadir%
mkdir %ship%\%datadir%
mkdir %ship%\%datadir%\logs
xcopy /e /y %PALETTE_SOURCE%\%datadir%\* %ship%\%datadir% >nul

rm -f "%ship%\%datadir%\saved\global\_Current.json"
rm -f "%ship%\%datadir%\saved\global\_Boot.json"

rem Attract-mode videos are a per-installation extra. The installer creates the
rem directory and its README, so the feature is discoverable and the engine has
rem somewhere to look, but it never ships the videos themselves: they are
rem hundreds of megabytes and whatever suits one venue rarely suits another.
rem The xcopy above brought along whatever videos this build machine happens to
rem have, so the directory is rebuilt here containing only the README.
rem An installed directory holding just the README does not turn the feature on;
rem the engine plays videos only when it finds actual video files.
rm -fr "%ship%\%datadir%\config\attractmode_videos"
if exist "%PALETTE_SOURCE%\%datadir%\config\attractmode_videos\README.md" (
	mkdir "%ship%\%datadir%\config\attractmode_videos"
	copy "%PALETTE_SOURCE%\%datadir%\config\attractmode_videos\README.md" "%ship%\%datadir%\config\attractmode_videos" >nul
)

echo ================ Creating installer for %datadir%

set "installer_output=%PALETTE_SOURCE%\release\palette_%version%_%datadir%.exe"
set "installer_delete=saved/quad/Filagree_Dance.json,saved/quad/Jigsaw_Puzzles.json,saved/quad/Pretty_Pulses.json,saved/quad/Too Many_Triangles.json"
call build_installer.bat data "%ship%\%datadir%" "%installer_output%" "%version%" "%data%" "%installer_delete%"
if errorlevel 1 goto getout
exit /b 0

:getout
exit /b 1

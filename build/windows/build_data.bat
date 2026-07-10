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

set ship=%PALETTE_SOURCE%\build\windows\ship
set datadir=data_%data%

rm -fr %ship% > nul 2>&1
mkdir %ship%

echo ================ Copying %datadir%
mkdir %ship%\%datadir%
mkdir %ship%\%datadir%\logs
xcopy /e /y %PALETTE_SOURCE%\%datadir%\* %ship%\%datadir% >nul

rm -f %ship%\%datadir%\saved\global\_Current.json
rm -f %ship%\%datadir%\saved\global\_Boot.json

echo ================ Creating installer for %datadir%

set "installer_output=%PALETTE_SOURCE%\release\palette_%version%_%datadir%.exe"
set "installer_delete=saved/quad/Filagree_Dance.json,saved/quad/Jigsaw_Puzzles.json,saved/quad/Pretty_Pulses.json,saved/quad/Too Many_Triangles.json"
call build_installer.bat data "%ship%\%datadir%" "%installer_output%" "%version%" "%data%" "%installer_delete%"
if errorlevel 1 goto getout

:getout

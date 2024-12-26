@echo off

set /p version=<../../VERSION
rem putting the PALETTE_VERSION in the environment so it can be used in the installer
set PALETTE_VERSION=%version%

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

set save_PALETTE_DATA=%PALETTE_DATA%
set PALETTE_DATA=%data%
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q data.iss

move Output\%datadir%_%version%.exe %PALETTE_SOURCE%\release\palette_%version%_%datadir%.exe >nul

rm -fr Output > nul 2>&1

:getout
set PALETTE_VERSION=
set PALETTE_DATA=%save_PALETTE_DATA%

@echo off

set /p version=<../../VERSION
rem putting the PALETTE_VERSION in the environment so it can be used in the installer
set PALETTE_VERSION=%version%

set data=%1

if not "%data%" == "" goto keepgoing0
	echo You must provide an argument, e.g. "omnisphere" or "sfmoma"
	goto getout
:keepgoing0

if not "%PALETTE_SOURCE%" == "" goto keepgoing1
	echo You must set the PALETTE_SOURCE environment variable.
	goto getout
:keepgoing1

set ship=%PALETTE_SOURCE%\build\windows\ship
set datadir=data_%data%

rm -fr %PALETTE_SOURCE%\%datadir%\logs
rm -fr %ship% > nul 2>&1
mkdir %ship%

echo ================ Copying %datadir%
mkdir %ship%\%datadir%
xcopy /e /y %PALETTE_SOURCE%\%datadir%\* %ship%\%datadir% >nul

echo ================ Creating installer for %datadir%

set PALETTE_DATA=%data%
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q data.iss

move Output\%datadir%_%version%.exe %PALETTE_SOURCE%\release\palette_%version%_%datadir%.exe >nul

rm -fr Output > nul 2>&1

:getout
set PALETTE_VERSION=

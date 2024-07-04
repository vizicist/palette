@echo off

set data=%PALETTE_DATA%

set /p version=<../../VERSION
set PALETTE_VERSION=%version%

set datadir=data_%data%

if not "%PALETTE_SOURCE%" == "" goto keepgoing1
	echo You must set the PALETTE_SOURCE environment variable.
	goto getout
:keepgoing1

set ship=%PALETTE_SOURCE%\build\windows\ship

rm -fr %ship% > nul 2>&1
mkdir %ship%

echo ================ Copying %datadir%
mkdir %ship%\%datadir%
xcopy /e /y %PALETTE_SOURCE%\%datadir%\* %ship%\%datadir% >nul

echo ================ Creating installer for %datadir%

"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q data_omnisphere.iss

move Output\%datadir%_%version%.exe %PALETTE_SOURCE%\release\palette_%version%_%datadir%.exe >nul

rm -fr Output > nul 2>&1

:getout

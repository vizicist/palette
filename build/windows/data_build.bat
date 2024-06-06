@echo off

set datadir=data_omnisphere

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

set /p version=<../../VERSION
sed -e "s/SUBSTITUTE_DATADIR_HERE/%datadir%/" < %datadir%.iss > tmp.iss
sed -e "s/SUBSTITUTE_VERSION_HERE/%version%/" < tmp.iss > tmp2.iss
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q tmp2.iss

move Output\%datadir%_%version%.exe %PALETTE_SOURCE%\release\palette_%version%_%datadir%.exe >nul

rem rm -fr Output > nul 2>&1
rem rm tmp.iss tmp2.iss

:getout

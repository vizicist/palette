@echo off

set data=%PALETTE_DATA%
if "%data%" == "" set data=omnisphere
set datadir=data_%data%

set /p version=<../../VERSION
echo ....................................................
echo Installing %datadir%_%version%
echo ....................................................

..\..\release\palette_%version%_%datadir%.exe /SILENT

echo ....................................................
echo Done!
echo ....................................................

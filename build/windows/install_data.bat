@echo off

set data=%PALETTE_DATA%
if "%data%" == "" set data=omnisphere
set datadir=data_%data%

set /p version=<../../VERSION
echo =============== Installing %datadir%_%version%

..\..\release\palette_%version%_%datadir%.exe /SILENT


@echo off

set datadir=data_omnisphere

set /p version=<../../VERSION
echo ....................................................
echo Installing %datadir%_%version%
echo ....................................................

..\..\release\%datadir%_%version%.exe /SILENT

echo ....................................................
echo Done!
echo ....................................................

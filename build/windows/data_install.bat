@echo off

set datadir=data_omnisphere

set /p version=<../../VERSION
echo ....................................................
echo Installing %datadir%_%version%
echo ....................................................

..\..\release\palette_%version%_%datadir%.exe /SILENT

echo ....................................................
echo Done!
echo ....................................................

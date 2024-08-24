@echo off
set /p version=<../../VERSION
hostname > hostname.txt
set /p hostname=<hostname.txt
del /q hostname.txt
echo You are about to install Palette version %version%
pause
..\..\release\palette_%version%_win_setup.exe /SILENT

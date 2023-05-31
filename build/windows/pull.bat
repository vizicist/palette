@echo off
set /p version=<../../VERSION
rm -f ..\..\release\palette_%version%_win_setup.exe
git pull

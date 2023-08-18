@echo off
set logdir=%CommonProgramFiles%\Palette\logs
cd %logdir%
del /s *.log >nul 2>nul
del /s *.stderr >nul 2>nul
del /s *.stdout >nul 2>nul
echo Logs in %logdir% have been cleared

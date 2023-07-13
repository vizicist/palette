@echo off
call setpalettelogdir
call sethostname
set logdir=%PALETTE_DATA_PATH%\logs\%hostname%
cd %logdir%
del /s *.log >nul 2>nul
del /s *.stderr >nul 2>nul
del /s *.stdout >nul 2>nul
echo Logs in %logdir% have been cleared

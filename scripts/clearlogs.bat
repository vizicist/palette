@echo off
call setpalettelogdir
del /s %PALETTE_LOGDIR%\*.log >nul 2>nul
del /s %PALETTE_LOGDIR%\*.stderr >nul 2>nul
del /s %PALETTE_LOGDIR%\*.stdout >nul 2>nul
echo Logs in %PALETTE_LOGDIR% have been cleared

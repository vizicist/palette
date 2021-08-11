@echo off
call setpalettelogdir
del /s %PALETTELOGDIR%\*.log >nul 2>nul
del /s %PALETTELOGDIR%\*.stderr >nul 2>nul
del /s %PALETTELOGDIR%\*.stdout >nul 2>nul
echo Logs in %PALETTELOGDIR% have been cleared

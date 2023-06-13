rem start /b palette_splash -image=sppro_rebooting.png -width=1920 -height=700
if exist %PALETTE_DATA_PATH%\config\tv_on.bat call %PALETTE_DATA_PATH%\config\tv_on.bat
rem bash checkmorphs.sh
palette restart
rem timeout /t 600 > nul
rem taskkill /f /im palette_splash.exe

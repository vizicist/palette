rem start /b palette_splash -image=sppro_rebooting.png -width=1920 -height=700
if exist %CommonProgramFiles%\Palette\config\tv_on.bat call %CommonProgramFiles%\Palette\config\tv_on.bat
rem bash checkmorphs.sh
palette start
rem timeout /t 600 > nul
rem taskkill /f /im palette_splash.exe

rem start /b palette_splash -image=sppro_rebooting.png -width=1920 -height=700
echo on

call cddata.bat
echo on
cd
if exist config\tv_on.bat call config\tv_on.bat

rem bash checkmorphs.sh
rem palette start
rem timeout /t 600 > nul
rem taskkill /f /im palette_splash.exe

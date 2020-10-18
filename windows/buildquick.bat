
@echo off

if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

set SHIPNAME=palette_win
set ship=%PALETTESOURCE%\ship\%SHIPNAME%
set bin=%ship%\bin

echo ================ COPYING scripts
pushd %PALETTESOURCE%\scripts
copy killall.bat %bin%
copy killpalette.bat %bin%
copy killgui.bat %bin%
copy killresolume.bat %bin%
copy killbidule.bat %bin%

copy startall.bat %bin%
copy startpalette.bat %bin%
copy startgui.bat %bin%
copy startresolume.bat %bin%
copy startbidule.bat %bin%

copy natsmon.bat %bin%
popd

echo ================ COPYING config

copy %PALETTESOURCE%\default\config\*.json %ship%\config
copy %PALETTESOURCE%\default\config\*.conf %ship%\config

:getout

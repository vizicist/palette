@echo off

pushd %PALETTE%

call bin\killall.bat

call bin\startpalette.bat
rem give NATS, etc time to finish starting
sleep 5
call bin\startgui.bat

rem If PALETTECONFIG is defined, then we assume we're doing bidule and/or resolume
if "%PALETTECONFIG%" == "" goto getout
sleep 2
call bin\startbidule.bat
sleep 5
call bin\startresolume.bat

:getout

popd

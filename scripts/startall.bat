@echo off

call %PALETTE%\bin\killall.bat

call %PALETTE%\bin\startpalette.bat
rem give NATS, etc time to finish starting
sleep 5
call %PALETTE%\bin\startgui.bat

if "%PALETTECONFIG%" == "" goto getout
sleep 2
call %PALETTE%\bin\startbidule.bat
sleep 5
call %PALETTE%\bin\startresolume.bat

:getout

@echo off

rem Provide an argument of "full" to 
if "%1" == "" goto defaultgui
set guitype=%1
goto keepgoing
:defaultgui
set guitype=viz
:keepgoing

set logdir=%LOCALAPPDATA%\Palette\logs

set gui=gui_%guitype%
echo > "%logdir%\%gui%.log"
echo > "%logdir%\%gui%.stdout"
echo > "%logdir%\%gui%.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\palette_%gui%.exe" > "%logdir%\%gui%.stdout" 2> "%logdir%\%gui%.stderr"

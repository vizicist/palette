@echo off
rem Provide an argument of "full" to 
if "%1" == "" goto remote
set guitype=%1
goto keepgoing
:remote
set guitype=remote
:keepgoing

c:/windows/system32/taskkill /F /IM palette_engine.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui_remote.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui_full.exe > nul 2>&1

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\engine.log"
echo > "%logdir%\engine.stdout"
echo > "%logdir%\engine.stderr"
start /b "" "%PALETTE%\bin\palette_engine.exe" > "%logdir%\engine.stdout" 2> "%logdir%\engine.stderr"

rem give NATS server a chance to start
sleep 4

set gui=gui_%guitype%
echo > "%logdir%\%gui%.log"
echo > "%logdir%\%gui%.stdout"
echo > "%logdir%\%gui%.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\palette_%gui%.exe" > "%logdir%\%gui%.stdout" 2> "%logdir%\%gui%.stderr"

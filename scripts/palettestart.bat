@echo off
rem Provide an argument of "full" to 
if "%1" == "" goto remote
set guitype=%1
goto keepgoing
:remote
set guitype=remote
:keepgoing

c:/windows/system32/taskkill /F /IM palette.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui_remote.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui_full.exe > nul 2>&1

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\palette.log"
echo > "%logdir%\palette.stdout"
echo > "%logdir%\palette.stderr"
start /b "" "%PALETTE%\bin\palette.exe" > "%logdir%\palette.stdout" 2> "%logdir%\palette.stderr"

rem give NATS server a chance to start
sleep 4

echo > "%logdir%\gui.log"
echo > "%logdir%\gui.stdout"
echo > "%logdir%\gui.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\palette_gui_%guitype%.exe" > "%logdir%\gui.stdout" 2> "%logdir%\gui.stderr"

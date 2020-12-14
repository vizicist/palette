@echo off

c:/windows/system32/taskkill /F /IM palette_gui.exe > nul 2>&1

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\gui.log"
echo > "%logdir%\gui.stdout"
echo > "%logdir%\gui.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\palette_gui.exe" > "%logdir%\gui.stdout" 2> "%logdir%\gui.stderr"

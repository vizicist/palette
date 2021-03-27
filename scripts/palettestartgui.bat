@echo off

c:/windows/system32/taskkill /F /IM palette_guiA.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_guiABCD.exe > nul 2>&1

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\gui.log"
echo > "%logdir%\gui.stdout"
echo > "%logdir%\gui.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\palette_guiA.exe" > "%logdir%\gui.stdout" 2> "%logdir%\gui.stderr"

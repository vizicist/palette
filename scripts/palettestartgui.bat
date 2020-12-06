@echo off

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\gui.log"
echo > "%logdir%\gui.stdout"
echo > "%logdir%\gui.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\palette_gui.exe" > "%logdir%\gui.stdout" 2> "%logdir%\gui.stderr"

@echo off

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\engine.log"
echo > "%logdir%\engine.stdout"
echo > "%logdir%\engine.stderr"
start /b "" "%PALETTE%\bin\palette_engine.exe" > "%logdir%\engine.stdout" 2> "%logdir%\engine.stderr"

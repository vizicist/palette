@echo off

c:/windows/system32/taskkill /F /IM palette_engine.exe > nul 2>&1

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\engine.log"
echo > "%logdir%\engine.stdout"
echo > "%logdir%\engine.stderr"
start /b "" "%PALETTE%\bin\palette_engine.exe" > "%logdir%\engine.stdout" 2> "%logdir%\engine.stderr"

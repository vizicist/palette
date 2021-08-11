@echo off

c:/windows/system32/taskkill /F /IM palette_engine.exe > nul 2>&1
call delay 1

call setpalettelogdir
echo > "%PALETTELOGDIR%\engine.log"
echo > "%PALETTELOGDIR%\engine.stdout"
echo > "%PALETTELOGDIR%\engine.stderr"
start /b "" "%PALETTE%\bin\palette_engine.exe" > "%PALETTELOGDIR%\engine.stdout" 2> "%PALETTELOGDIR%\engine.stderr"

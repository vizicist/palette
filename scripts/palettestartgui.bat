@echo off

c:/windows/system32/taskkill /F /IM palette_gui.exe > nul 2>&1

call setpalettelogdir

echo > "%PALETTELOGDIR%\gui.log"
echo > "%PALETTELOGDIR%\gui.stdout"
echo > "%PALETTELOGDIR%\gui.stderr"

start /b "" "%PALETTE%\bin\pyinstalled\palette_gui.exe" > "%PALETTELOGDIR%\gui.stdout" 2> "%PALETTELOGDIR%\gui.stderr"

@echo off

c:/windows/system32/taskkill /F /IM palette_guiA.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_guiABCD.exe > nul 2>&1

call setpalettelogdir
if not "%PALETTELOGDIR%" == "" (
	echo > "%PALETTELOGDIR%\gui.log"
	echo > "%PALETTELOGDIR%\gui.stdout"
	echo > "%PALETTELOGDIR%\gui.stderr"
	start /b "" "%PALETTE%\bin\pyinstalled\palette_guiA.exe" > "%PALETTELOGDIR%\gui.stdout" 2> "%PALETTELOGDIR%\gui.stderr"
)

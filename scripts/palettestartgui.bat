@echo off

c:/windows/system32/taskkill /F /IM palette_gui.exe > nul 2>&1

call setpalettelogdir

if not "%PALETTELOGDIR%" == "" (
	echo > "%PALETTELOGDIR%\gui.log"
	echo > "%PALETTELOGDIR%\gui.stdout"
	echo > "%PALETTELOGDIR%\gui.stderr"
)

if not "%PALETTESOURCE%" == "" (
	start /b "" "%PALETTESOURCE%\build\windows\ship\bin\pyinstalled\palette_gui.exe" > "%PALETTELOGDIR%\gui.stdout" 2> "%PALETTELOGDIR%\gui.stderr"
) else (
	start /b "" "%PALETTE%\bin\pyinstalled\palette_gui.exe" > "%PALETTELOGDIR%\gui.stdout" 2> "%PALETTELOGDIR%\gui.stderr"
)
call delay 2
call resizegui

@echo off

call setpalettelogdir

if exist "C:\\Program Files\\Resolume Avenue\\Avenue.exe" (
	set exe="Avenue.exe"
	set res="C:\\Program Files\\Resolume Avenue\\Avenue.exe"
) else if exist "C:\\Program Files\\Resolume Arena\\Arena.exe" (
	set exe="Arena.exe"
	set res="C:\\Program Files\\Resolume Arena\\Arena.exe"
) else (
	echo No Resolume Avenue 7 or Arena 7 found!
	goto getout:
)

c:/windows/system32/taskkill /F /IM %exe% >nul 2>&1
rem Give it time to stop, otherwise start says process busy
call delay 1
start /b "" %res% > "%PALETTELOGDIR%\resolume.log" 2>&1

call delay 7
call paletteactivateresolume

:getout

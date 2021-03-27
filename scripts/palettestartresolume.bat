@echo off

call setpalettelogdir
if "%PALETTELOGDIR%" == "" (
	echo Unable to set PALETTELOGDIR
	exit
)

if exist "C:\\Program Files\\Resolume Avenue\\Avenue.exe" (

	c:/windows/system32/taskkill /F /IM Avenue.exe >nul 2>&1
	start /b "" "C:\\Program Files\\Resolume Avenue\\Avenue.exe" > "%PALETTELOGDIR%\resolume.log" 2>&1

) else if exist "C:\\Program Files\\Resolume Arena\\Arena.exe" (

	c:/windows/system32/taskkill /F /IM Arena.exe >nul 2>&1
	start /b "" "C:\\Program Files\\Resolume Arena\\Arena.exe" > "%PALETTELOGDIR%\resolume.log" 2>&1

) else (
	echo No Resolume Avenue 7 or Arena 7 found!
	goto getout:
)

rem give it time to start before sending it OSC
call delay 4

set osc="%PALETTE%\bin\pyinstalled\osc.exe"
for /f %%i in ('ipaddress') do set addr=%%i
set port=7000

for %%g in (a,b,c,d,e,f,g,h,i) do (
	call delay 3
	echo Sending OSC to activate Resolume
	%osc% send %port%@%addr% /composition/layers/1/clips/1/connect 1
	%osc% send %port%@%addr% /composition/layers/2/clips/1/connect 1
	%osc% send %port%@%addr% /composition/layers/3/clips/1/connect 1
	%osc% send %port%@%addr% /composition/layers/4/clips/1/connect 1
)

:getout

@echo off

call palettestopresolume.bat

rem give it time to start so it closes the log file
call delay 1
if exist "C:\\Program Files\\Resolume Avenue\\Avenue.exe" (
	start /b "" "C:\\Program Files\\Resolume Avenue\\Avenue.exe" > "%LOCALAPPDATA%\Palette\logs\resolume.log" 2>&1
) else if exist "C:\\Program Files\\Resolume Arena\\Arena.exe" (
	start /b "" "C:\\Program Files\\Resolume Arena\\Arena.exe" > "%LOCALAPPDATA%\Palette\logs\resolume.log" 2>&1
) else (
	echo No Resolume Avenue 7 or Arena 7 found!
	goto getout:
)

rem give it time to start before sending it OSC
call delay 4

set osc="%PALETTE%\bin\pyinstalled\osc.exe"
for /f %%i in ('ipaddress') do set addr=%%i
set port=7000

%osc% send %port%@%addr% /composition/layers/1/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/2/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/3/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/4/clips/1/connect 1

rem another try in case Resolume takes longer to start
call delay 4

%osc% send %port%@%addr% /composition/layers/1/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/2/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/3/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/4/clips/1/connect 1

:getout

@echo off

call palettestopresolume6.bat
start /b "" "C:\\Program Files\\Resolume Avenue 6\\Avenue.exe"

rem give it time to start before sending it OSC
timeout /t 4 > nul

set osc="%PALETTE%\bin\pyinstalled\osc.exe"
for /f %%i in ('ipaddress') do set addr=%%i
set port=7000

%osc% send %port%@%addr% /composition/layers/1/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/2/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/3/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/4/clips/1/connect 1

rem another try in case Resolume takes longer to start
timeout /t 4 > nul

%osc% send %port%@%addr% /composition/layers/1/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/2/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/3/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/4/clips/1/connect 1

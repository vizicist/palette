@echo off

call palettestopresolume.bat
start /b "" "C:\\Program Files\\Resolume Avenue 6\\Avenue.exe"

rem give it time to start before sending it OSC
sleep 8

set osc="%PALETTE%\bin\pyinstalled\osc.exe"
%osc% send 7000@127.0.0.1 /composition/layers/1/clips/1/connect 1
%osc% send 7000@127.0.0.1 /composition/layers/2/clips/1/connect 1
%osc% send 7000@127.0.0.1 /composition/layers/3/clips/1/connect 1
%osc% send 7000@127.0.0.1 /composition/layers/4/clips/1/connect 1

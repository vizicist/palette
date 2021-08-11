@echo off

set osc="%PALETTE%\bin\pyinstalled\osc.exe"
set addr=127.0.0.1

rem Activate Resolume layers
set port=7000
%osc% send %port%@%addr% /composition/layers/1/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/2/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/3/clips/1/connect 1
%osc% send %port%@%addr% /composition/layers/4/clips/1/connect 1

rem Activate Bidule audio
set port=3210
%osc% send %port%@%addr% /play 1

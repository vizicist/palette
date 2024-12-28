@echo off

if not "%PALETTE_DATA%" == "" goto keepgoing1
	set PALETTE_DATA=default
:keepgoing1

set datapath=C:\\Program Files\\Common Files\\Palette\\data_%PALETTE_DATA%

cd %datapath%

:out

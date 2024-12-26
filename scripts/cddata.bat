@echo off

if not "%PALETTE_DATA%" == "" goto keepgoing1
	set PALETTE_DATA=default
:keepgoing1

if not "%PALETTE_SOURCE%" == "" goto usesource
	set datapath=C:\\Program Files\\Common Files\\Palette\\data_%PALETTE_DATA%
	goto keepgoing2
:usesource
	set datapath="%PALETTE_SOURCE%\\data_%PALETTE_DATA%"
:keepgoing2

cd %datapath%

:out

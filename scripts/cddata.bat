@echo off

if not "%PALETTE_DATA%" == "" goto keepgoing1
	set PALETTE_DATA=default
:keepgoing1

set datapath=%LOCALAPPDATA%\Palette\data_%PALETTE_DATA%
if not "%PALETTE_DATAROOT%" == "" set datapath=%PALETTE_DATAROOT%\data_%PALETTE_DATA%

if "%1" == "" (
	pushd "%datapath%"
) else (
	pushd "%datapath%\%1"
)

:out

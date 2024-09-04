@echo off

if not "%PALETTE_DATA_PATH%" == "" goto keepgoing
	echo You need to set PALETTE_DATA_PATH!
	goto out

:keepgoing

cd "C:\\Program Files\\Common Files\\Palette\\data_%PALETTE_DATA%"

:out

@echo off

set data=%1
if not "%data%" == "" goto keepgoing0
	echo You must provide an argument, e.g. "omnisphere" or "sfmoma"
	goto getout
:keepgoing0

set datadir=data_%data%

set /p version=<../../VERSION
echo =============== Installing %datadir%_%version%

..\..\release\palette_%version%_%datadir%.exe /SILENT

:getout

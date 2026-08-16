@echo off

if not "%PALETTE_SOURCE%" == "" goto keepgoing
echo You must set the PALETTE_SOURCE environment variable.
goto getout

:keepgoing

rem Refuse to run against anything that is not a Palette source tree.
rem Every destructive path below is built from these variables, so an empty
rem or mistaken PALETTE_SOURCE aims them somewhere else entirely.
if not exist "%PALETTE_SOURCE%\VERSION" (
	echo ERROR: PALETTE_SOURCE does not look like a Palette source tree:
	echo     %PALETTE_SOURCE%
	exit /b 1
)

rm -fr "%PALETTE_SOURCE%\ffgl\build\windows\.vs"
rm -fr "%PALETTE_SOURCE%\ffgl\build\windows\x64"
rm -fr "%PALETTE_SOURCE%\ffgl\build\windows\x86"
rm -fr "%PALETTE_SOURCE%\build\windows\ship"
rm -fr "%PALETTE_SOURCE%\python\build"
rm -fr "%PALETTE_SOURCE%\python\dist"
rm -fr "%PALETTE_SOURCE%\depthlib\build\.vs"
rm -fr "%PALETTE_SOURCE%\depthlib\build\x64"
rm -fr "%PALETTE_SOURCE%\depthlib\build\*.dir"
del /s "%PALETTE_SOURCE%\mmtt_kinect\build\windows\*.obj" >nul 2>nul
del /s "%PALETTE_SOURCE%\data_default\logs\*.log" >nul 2>nul
del /s "%PALETTE_SOURCE%\data_cloud\logs\*.log" >nul 2>nul

:getout

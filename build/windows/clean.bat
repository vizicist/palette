@echo off

if not "%PALETTE_SOURCE%" == "" goto keepgoing
echo You must set the PALETTE_SOURCE environment variable.
goto getout

:keepgoing

rm -fr %PALETTE_SOURCE%\ffgl\build\windows\.vs
rm -fr %PALETTE_SOURCE%\ffgl\build\windows\x64
rm -fr %PALETTE_SOURCE%\ffgl\build\windows\x86
rm -fr %PALETTE_SOURCE%\build\windows\ship
rm -fr %PALETTE_SOURCE%\python\build
rm -fr %PALETTE_SOURCE%\python\dist
rm -fr %PALETTE_SOURCE%\depthlib\build\.vs
rm -fr %PALETTE_SOURCE%\depthlib\build\x64
rm -fr %PALETTE_SOURCE%\depthlib\build\*.dir
del /s %PALETTE_SOURCE%\mmtt_kinect\build\windows\*.obj >nul 2>nul
del /s %PALETTE_SOURCE%\data_sfmoma\logs\*.log >nul 2>nul
del /s %PALETTE_SOURCE%\data_omnisphere\logs\*.log >nul 2>nul

:getout

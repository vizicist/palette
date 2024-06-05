@echo off

set data="omnisphere"
if not "%PALETTE_DATA%" == "" set data=%PALETTE_DATA

if "%PALETTE_SOURCE%" == "" goto usecommon
cd "%PALETTE_SOURCE%\\data_%PALETTE_DATA%\logs"
goto out

:usecommon
cd "C:\\Program Files\\Common Files\\Palette\\data_%PALETTE_DATA%\\logs"

:out

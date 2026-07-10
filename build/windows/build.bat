@echo off

echo BUILDING binaries...
call build_bin
if errorlevel 1 exit /b %ERRORLEVEL%

echo BUILDING data_default...
call build_data default
exit /b %ERRORLEVEL%

@echo off
call build.bat
if errorlevel 1 exit /b %ERRORLEVEL%
call install.bat
if errorlevel 1 exit /b %ERRORLEVEL%
call run.bat
exit /b %ERRORLEVEL%

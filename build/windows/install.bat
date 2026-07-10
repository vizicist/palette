@echo off
echo STOPPING...
palette stop 2>nul
echo INSTALLING...
call install_bin
if errorlevel 1 exit /b %ERRORLEVEL%
call install_data default
exit /b %ERRORLEVEL%

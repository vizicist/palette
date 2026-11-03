@echo off
setlocal
pushd "%~dp0"
if errorlevel 1 exit /b %ERRORLEVEL%

call :step Testing testit.bat doit-test.out
if errorlevel 1 goto failed
call :step Building build.bat doit-build.out
if errorlevel 1 goto failed
call :step Installing install.bat doit-install.out
if errorlevel 1 goto failed
call :step Starting run.bat doit-run.out
if errorlevel 1 goto failed

echo Done. Details: %~dp0doit-*.out
popd
exit /b 0

:step
echo %~1 Palette...
set "DOIT_LOG=%~dp0%~3"
call "%~dp0%~2" >"%DOIT_LOG%" 2>&1
set "DOIT_RESULT=%ERRORLEVEL%"
if "%DOIT_RESULT%" == "0" exit /b 0
echo %~1 failed. Log: %DOIT_LOG%
type "%DOIT_LOG%"
exit /b %DOIT_RESULT%

:failed
set "DOIT_RESULT=%ERRORLEVEL%"
popd
exit /b %DOIT_RESULT%

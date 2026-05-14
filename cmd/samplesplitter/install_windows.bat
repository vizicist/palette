@echo off
setlocal EnableExtensions

set "PY_VERSION=3.11"
set "PY_CMD=py -%PY_VERSION%"
set "MODULES=pyo mido python-rtmidi"
set "PAUSE_ON_EXIT=1"
set "WINGET_EXTRA_FLAGS="

:parse_args
if "%~1" == "" goto args_done
if /i "%~1" == "--quiet" set "PAUSE_ON_EXIT=0"
if /i "%~1" == "--no-pause" set "PAUSE_ON_EXIT=0"
if /i "%~1" == "/quiet" set "PAUSE_ON_EXIT=0"
if /i "%~1" == "/silent" set "PAUSE_ON_EXIT=0"
if /i "%~1" == "/verysilent" set "PAUSE_ON_EXIT=0"
shift
goto parse_args

:args_done
if "%PAUSE_ON_EXIT%" == "0" set "WINGET_EXTRA_FLAGS=--silent"

echo Sample Splitter Windows installer
echo.

where py >nul 2>nul
if errorlevel 1 (
    echo Python Launcher was not found.
    call :install_python
) else (
    %PY_CMD% -c "import sys" >nul 2>nul
    if errorlevel 1 (
        echo Python %PY_VERSION% was not found.
        call :install_python
    )
)

echo.
echo Checking Python %PY_VERSION%...
%PY_CMD% -c "import sys; print(sys.executable); print(sys.version)" 2>nul
if errorlevel 1 (
    echo.
    echo Python %PY_VERSION% is still not available through the Python Launcher.
    echo Close this window, open a new Command Prompt, and run this installer again.
    goto :fail
)

echo.
echo Ensuring pip is available...
%PY_CMD% -m ensurepip --upgrade
if errorlevel 1 goto :fail

echo.
echo Upgrading pip...
%PY_CMD% -m pip install --user --upgrade pip
if errorlevel 1 goto :fail

echo.
echo Installing Python modules: %MODULES%
%PY_CMD% -m pip install --user %MODULES%
if errorlevel 1 goto :fail

echo.
echo Verifying imports...
%PY_CMD% -c "import pyo, mido, rtmidi; print('pyo, mido, and python-rtmidi are installed.')"
if errorlevel 1 goto :fail

echo.
echo Done. You can now run runit.bat.
if "%PAUSE_ON_EXIT%" == "1" pause
exit /b 0

:install_python
echo.
where winget >nul 2>nul
if errorlevel 1 (
    echo winget was not found, so this script cannot install Python automatically.
    echo Install Python %PY_VERSION% from https://www.python.org/downloads/windows/
    echo Make sure "py launcher" is selected, then run this installer again.
    goto :fail
)

echo Installing Python %PY_VERSION% with winget...
winget install --id Python.Python.%PY_VERSION% -e --source winget --accept-package-agreements --accept-source-agreements %WINGET_EXTRA_FLAGS%
if errorlevel 1 goto :fail
exit /b 0

:fail
echo.
echo Install failed with code %ERRORLEVEL%.
if "%PAUSE_ON_EXIT%" == "1" pause
exit /b 1

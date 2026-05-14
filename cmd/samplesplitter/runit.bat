@echo off
setlocal
pushd "%~dp0"
set "PORT=9876"

py -3.11 -c "import sys" >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    set "PY_CMD=py -3.11"
) else (
    echo Python 3.11 was not found. Falling back to the default Python 3 runtime.
    echo For audio playback, install Python 3.11 and then run:
    echo   py -3.11 -m pip install --user pyo mido python-rtmidi
    echo.
    set "PY_CMD=py -3"
)

echo Stopping any existing Sample Splitter servers...
powershell -NoProfile -ExecutionPolicy Bypass -Command "Get-CimInstance Win32_Process | Where-Object { $_.ProcessId -ne $PID -and $_.CommandLine -and $_.CommandLine -match 'samplesplitter\.py' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }" >nul 2>nul
timeout /t 1 /nobreak >nul

echo Starting Sample Splitter on http://localhost:%PORT%/
start "Sample Splitter Server" cmd /k "%PY_CMD% samplesplitter.py --port %PORT%"

echo Waiting for the web interface...
for /l %%i in (1,1,30) do (
    powershell -NoProfile -ExecutionPolicy Bypass -Command "try { $r = Invoke-WebRequest -UseBasicParsing -Uri 'http://127.0.0.1:%PORT%/' -TimeoutSec 1; if ($r.StatusCode -ge 200) { exit 0 } else { exit 1 } } catch { exit 1 }" >nul 2>nul
    if not errorlevel 1 goto open_ui
    timeout /t 1 /nobreak >nul
)

echo The server did not respond on http://localhost:%PORT%/ within 30 seconds.
echo Check the "Sample Splitter Server" window for errors.
pause
popd
exit /b 1

:open_ui
start "" "http://localhost:%PORT%/"
popd
endlocal

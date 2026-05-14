@echo off
setlocal

set "PALETTE_EXE=%LOCALAPPDATA%\Programs\Palette\bin\palette.exe"

if exist "%PALETTE_EXE%" goto run_installed
where palette.exe >nul 2>nul
if errorlevel 1 goto not_found
set "PALETTE_EXE=palette.exe"

:run_installed
if "%~1" == "" (
	echo RUNNING Palette...
	"%PALETTE_EXE%" start
) else (
	echo RUNNING Palette %*...
	"%PALETTE_EXE%" start %*
)
goto done

:not_found
echo Unable to find palette.exe.
echo Expected it at "%LOCALAPPDATA%\Programs\Palette\bin\palette.exe".
exit /b 1

:done
endlocal

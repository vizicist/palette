@echo off
setlocal

set PALETTE_SOURCE=%~dp0..\..
for %%I in ("%PALETTE_SOURCE%") do set PALETTE_SOURCE=%%~fI

if "%PALETTE%" == "" (
    set PALETTE_INSTALL=C:\Program Files\Palette
) else (
    set PALETTE_INSTALL=%PALETTE%
)

echo STOPPING Palette...
palette stop 2>nul

echo BUILDING palette_engine...
call "C:\Program Files\Microsoft Visual Studio\18\Community\VC\Auxiliary\Build\vcvars64.bat" >nul
set "MINGW64_BIN=%USERPROFILE%\mingw64\bin"
if not exist "%MINGW64_BIN%\gcc.exe" (
    echo MinGW-w64 gcc.exe was not found under %MINGW64_BIN%.
    exit /b 1
)
set "PATH=%MINGW64_BIN%;%PATH%"
set CGO_ENABLED=1
pushd "%PALETTE_SOURCE%\cmd\palette_engine"
go build -o "%PALETTE_INSTALL%\bin\palette_engine.exe" palette_engine.go
set BUILD_RESULT=%ERRORLEVEL%
popd

if not "%BUILD_RESULT%" == "0" (
    echo palette_engine build failed.
    exit /b %BUILD_RESULT%
)

echo RESTARTING Palette...
palette restart

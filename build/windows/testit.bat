@echo off
setlocal

set PALETTE_SOURCE=%~dp0..\..
for %%I in ("%PALETTE_SOURCE%") do set PALETTE_SOURCE=%%~fI

echo TESTING Palette...

echo ================ Web UI smoke test
rem NODE_EXE may point to a portable Node installation outside PATH.
if defined NODE_EXE goto node_ready
for /f "delims=" %%N in ('where node.exe 2^>nul') do if not defined NODE_EXE set "NODE_EXE=%%N"
if defined NODE_EXE goto node_ready
for %%N in ("%ProgramFiles%\nodejs\node.exe" "%ProgramFiles(x86)%\nodejs\node.exe" "%LOCALAPPDATA%\Programs\nodejs\node.exe") do if not defined NODE_EXE if exist "%%~N" set "NODE_EXE=%%~N"
if defined NODE_EXE goto node_ready
rem Use Codex's bundled runtime when no standalone Node installation is available.
if exist "%USERPROFILE%\.cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin\node.exe" set "NODE_EXE=%USERPROFILE%\.cache\codex-runtimes\codex-primary-runtime\dependencies\node\bin\node.exe"
if defined NODE_EXE goto node_ready
echo ERROR: Node.js is required to run the Web UI smoke test.
echo Install Node.js and reopen your terminal, or set NODE_EXE to the full path to node.exe.
exit /b 1

:node_ready
"%NODE_EXE%" "%PALETTE_SOURCE%\build\webui_smoke_test.mjs" %*
if errorlevel 1 exit /b %ERRORLEVEL%

echo ================ Go kit tests
pushd "%PALETTE_SOURCE%"
rem Keep regression tests scoped to runtime packages. Do not include cmd/miditest;
rem it is a manual hardware/MIDI diagnostic command, not a regression target.
go test ./kit
set KIT_RESULT=%ERRORLEVEL%
popd
if not "%KIT_RESULT%" == "0" exit /b %KIT_RESULT%

echo ================ SampleSplitter tests
pushd "%PALETTE_SOURCE%"
go test ./pkg/samplesplitter
set SS_RESULT=%ERRORLEVEL%
popd
if not "%SS_RESULT%" == "0" exit /b %SS_RESULT%

echo TESTS PASSED
endlocal

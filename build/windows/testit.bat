@echo off
setlocal

set PALETTE_SOURCE=%~dp0..\..
for %%I in ("%PALETTE_SOURCE%") do set PALETTE_SOURCE=%%~fI

echo TESTING Palette...

echo ================ Web UI smoke test
node "%PALETTE_SOURCE%\build\webui_smoke_test.mjs" %*
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

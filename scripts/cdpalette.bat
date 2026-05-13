@echo off

set PALETTE_APP=%LOCALAPPDATA%\Programs\Palette
if not "%PALETTE%" == "" set PALETTE_APP=%PALETTE%

pushd "%PALETTE_APP%"

@echo off

taskkill /f /im palette_engine.exe
REM Close only the Palette Control Chrome window, not all Chrome windows
powershell -Command "Get-Process chrome -ErrorAction SilentlyContinue | Where-Object {$_.MainWindowTitle -eq 'Palette Control'} | ForEach-Object { $_.CloseMainWindow() | Out-Null }"
taskkill /f /im palette_monitor.exe
taskkill /f /im palette_splash.exe
taskkill /f /im key.exe
taskkill /f /im mmtt_kinect.exe
taskkill /f /im avenue.exe
taskkill /f /im arena.exe
taskkill /f /im bidule.exe
taskkill /f /im palette.exe

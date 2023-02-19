@echo off
echo Before:
tasklist | grep palette

taskkill /f /im palette_engine.exe
taskkill /f /im palette_gui.exe
taskkill /f /im key.exe

echo After:
tasklist | grep palette

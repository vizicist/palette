@echo off

echo Palette restart underway

taskkill /f /im palette_engine.exe
taskkill /f /im palette_gui.exe
taskkill /f /im palette_monitor.exe
taskkill /f /im key.exe
taskkill /f /im bidule.exe
taskkill /f /im avenue.exe
taskkill /f /im arena.exe

echo Starting monitor

start /min palette_monitor

timeout -T 5 > nul

echo Starting the rest

palette start gui
palette start bidule
palette start resolume


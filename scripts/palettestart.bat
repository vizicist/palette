@echo off

c:/windows/system32/taskkill /F /IM palette_engine.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui.exe > nul 2>&1

call palettestartengine
call palettestartgui

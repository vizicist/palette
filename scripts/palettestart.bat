@echo off

rem Provide an argument of "full" to 
if "%1" == "" goto defaultgui
set guitype=%1
goto keepgoing
:defaultgui
set guitype=viz
:keepgoing

c:/windows/system32/taskkill /F /IM palette_engine.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui_remote.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui_viz.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM palette_gui_full.exe > nul 2>&1

call palettestartengine
call palettestartgui %guitype%

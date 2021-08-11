@echo off
echo Stopping everything
call palettestopall
call delay 3

echo Starting Resolume
call palettestartresolume

echo Starting Bidule
call palettestartbidule

echo delaying 10
call delay 10

echo Starting Engine
call palettestartengine

echo Starting GUI
call palettestartgui
call delay 4
echo Resizing GUI
call resizegui

call paletteactivate

@echo off
vsinstr.exe loopycam.exe
del /q loopycam.vsp
vsperfcmd /start:trace /output:loopycam.vsp
echo Run loopycam now

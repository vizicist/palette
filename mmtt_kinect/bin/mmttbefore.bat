@echo off
vsinstr.exe mmtt_kinect.exe
del /q mmtt_kinect.vsp
vsperfcmd /start:trace /output:mmtt_kinect.vsp
echo Run mmtt_kinect now

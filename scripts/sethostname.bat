@echo off
hostname > hostname.txt
set /p hostname=<hostname.txt
del /q hostname.txt

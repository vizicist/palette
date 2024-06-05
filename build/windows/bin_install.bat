@echo off
set /p version=<../../VERSION
hostname > hostname.txt
set /p hostname=<hostname.txt
del /q hostname.txt
echo ....................................................
echo Installing Palette version %version%
echo ....................................................
if not "%PALETTE_MMTT%" == "kinect" goto no_mmtt_kinect
..\..\release\palette_%version%_win_setup_with_kinect.exe /SILENT
goto done
:no_mmtt_kinect
..\..\release\palette_%version%_win_setup.exe /SILENT
:done
echo ....................................................
echo Done!
echo ....................................................

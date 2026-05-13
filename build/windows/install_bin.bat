@echo off
set /p version=<../../VERSION
hostname > hostname.txt
set /p hostname=<hostname.txt
del /q hostname.txt
echo =============== Installing Palette version %version%
if not "%PALETTE_MMTT%" == "kinect" goto no_mmtt_kinect
..\..\release\palette_%version%_win_setup_with_kinect.exe /CURRENTUSER /VERYSILENT /SUPPRESSMSGBOXES
goto done
:no_mmtt_kinect
..\..\release\palette_%version%_win_setup.exe /CURRENTUSER /VERYSILENT /SUPPRESSMSGBOXES
:done

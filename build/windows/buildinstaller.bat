@echo off

copy %PALETTE_SOURCE%\VERSION %ship% >nul
set /p version=<../../VERSION

echo ================ Creating installer for VERSION %version%

sed -e "s/SUBSTITUTE_VERSION_HERE/%version%/" < palette_win_setup.iss > tmp.iss
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q tmp.iss

if not "%PALETTE_MMTT%" == "kinect" goto no_kinect
move Output\palette_%version%_win_setup.exe %PALETTE_SOURCE%\release\palette_%version%_win_setup_with_kinect.exe >nul
goto finish

:no_kinect
move Output\palette_%version%_win_setup.exe %PALETTE_SOURCE%\release >nul

:finish

rmdir Output
rm tmp.iss

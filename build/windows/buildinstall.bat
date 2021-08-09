call build.bat
call palettestopall
set INSTALLER=..\..\release\palette_5.1_win_setup.exe
copy %INSTALLER% t:\tjt\media\spacepalettepro\installers >nul
%INSTALLER%

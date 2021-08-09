call build.bat
call palettestopall
set INSTALLER=..\..\release\palette_5.11_win_setup.exe
copy %INSTALLER% t:\tjt\media\spacepalettepro\installers >nul
%INSTALLER%

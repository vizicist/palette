call palette stop all
call build.bat
set /p version=<../../VERSION
set INSTALLER=..\..\release\palette_%version%_win_setup.exe
copy %INSTALLER% t:\tjt\media\spacepalettepro\installers >nul
%INSTALLER%

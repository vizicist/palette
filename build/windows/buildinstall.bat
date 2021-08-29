call build.bat
call palette stop all
set /p version=<../../VERSION
set INSTALLER=..\..\release\palette_%version%_win_setup.exe
copy %INSTALLER% t:\tjt\media\spacepalettepro\installers >nul
%INSTALLER%

rem call palette stop
call build.bat
set /p version=<../../VERSION
set INSTALLER=..\..\release\palette_%version%_win_setup.exe
rem copy %INSTALLER% t:\tjt\media\spacepalettepro\installers >nul
%INSTALLER%

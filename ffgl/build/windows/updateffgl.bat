call ..\..\..\scripts\killresolume.bat
call delay 3
echo on
set FFGL="c:\Program Files\Palette\ffgl"
if "%PALETTE_DATA%" == "" set PALETTE_DATA=default
set PALETTE_CONFIG=%LOCALAPPDATA%\Palette\data_%PALETTE_DATA%\config
copy ..\..\source\lib\pthreads\Pre-built.2\dll\x64\pthreadVC2.dll %FFGL%
rem copy ..\..\..\build\windows\msvcr100.dll %FFGL%
copy ..\..\binaries\x64\Debug\Palette.dll %FFGL%
copy ..\..\binaries\x64\Debug\Palette.pdb %FFGL%
copy ..\..\..\data_omnisphere\config\paramdefs.json "%PALETTE_CONFIG%\paramdefs.json"
copy ..\..\..\data_omnisphere\config\resolume.json "%PALETTE_CONFIG%\resolume.json"
copy ..\..\..\data_omnisphere\config\synths.json "%PALETTE_CONFIG%\synths.json"
palette start resolume

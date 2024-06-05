call ..\..\..\scripts\killresolume.bat
call delay 3
echo on
set FFGL="c:\Program Files\Palette\ffgl"
copy ..\..\source\lib\pthreads\Pre-built.2\dll\x64\pthreadVC2.dll %FFGL%
rem copy ..\..\..\build\windows\msvcr100.dll %FFGL%
copy ..\..\binaries\x64\Debug\Palette.dll %FFGL%
copy ..\..\binaries\x64\Debug\Palette.pdb %FFGL%
copy ..\..\..\data_omnisphere\config\paramdefs.json "%CommonProgramFiles%\Palette\data_%PALETTE_DATA%\config\paramdefs.json"
copy ..\..\..\data_omnisphere\config\resolume.json "%CommonProgramFiles%\Palette\data_%PALETTE_DATA%\config\resolume.json"
copy ..\..\..\data_omnisphere\config\synths.json "%CommonProgramFiles%\Palette\data_%PALETTE_DATA%\config\synths.json"
palette start resolume

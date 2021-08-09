call palettestopresolume
call delay 3
copy ..\..\source\lib\pthreads\Pre-built.2\dll\x64\pthreadVC2.dll "c:\Program Files\Palette\ffgl"
copy ..\..\..\build\windows\msvcr100.dll "c:\Program Files\Palette\ffgl"
copy ..\..\binaries\x64\Debug\Palette.dll "c:\Program Files\Palette\ffgl"
copy ..\..\binaries\x64\Debug\Palette.pdb "c:\Program Files\Palette\ffgl"
copy ..\..\..\default\config\paramdefs.json "%LOCALAPPDATA%\Palette\config\paramdefs.json"
copy ..\..\..\default\config\resolume.json "%LOCALAPPDATA%\Palette\config\resolume.json"
copy ..\..\..\default\config\synths.json "%LOCALAPPDATA%\Palette\config\synths.json"
rem call palettestartresolume7

call palettestopresolume
timeout /t 3 > nul
copy ..\..\binaries\x64\Debug\Palette.dll "c:\Program Files\Palette\ffgl"
copy ..\..\binaries\x64\Debug\Palette.pdb "c:\Program Files\Palette\ffgl"
copy ..\..\..\default\config\paramdefs.json "c:\Program Files\Palette\config\paramdefs.json"
copy ..\..\..\default\config\resolume.json "c:\Program Files\Palette\config\resolume.json"
copy ..\..\..\default\config\synths.json "c:\Program Files\Palette\config\synths.json"
rem call palettestartresolume7

echo stopping resolume
call palettestopresolume
echo calling delay
call delay 2
echo copying files
copy ..\..\binaries\x64\Debug\Palette*.dll "c:\Program Files\Palette\ffgl"
copy ..\..\binaries\x64\Debug\Palette*.pdb "c:\Program Files\Palette\ffgl"
copy ..\..\..\default\config\paramdefs.json "c:\Program Files\Palette\config\paramdefs.json"
copy ..\..\..\default\config\resolume.json "c:\Program Files\Palette\config\resolume.json"
copy ..\..\..\default\config\synths.json "c:\Program Files\Palette\config\synths.json"
echo starting resolume
rem call palettestartresolume

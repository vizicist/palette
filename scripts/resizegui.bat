set name="Palette ABCD"
nircmdc.exe win setsize stitle %name% -800 0 800 1280
rem nircmdc.exe win setsize stitle %name% -800 0 800 1280
nircmdc.exe win -style stitle %name% 0x00CA0000
nircmdc.exe win max stitle %name%

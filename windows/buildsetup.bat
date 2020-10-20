echo ================ CREATING palette_setup_win.exe
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" palette_win_setup.iss
move Output\palette_win_setup.exe .
rmdir Output

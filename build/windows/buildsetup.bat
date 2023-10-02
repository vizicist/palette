echo ================ Creating installer
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q palette_win_setup.iss
move Output\palette_*_win_setup.exe %PALETTE_SOURCE%\release >nul
rmdir Output

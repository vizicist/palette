@echo off
if "%PALETTE_MMTT%" == "" set PALETTE_MMTT=none
echo STOPPING...
palette stop
echo INSTALLING...
call install_bin
call install_data sfmoma

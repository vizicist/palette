@echo off
palette stop

set save_PALETTE_MMTT=%PALETTE_MMTT%

echo "====================== creating build with PALETTE_MMTT not set"
set PALETTE_MMTT=
call clean.bat
call build.bat
echo "====================== creating build with PALETTE_MMTT set to kinect"
set PALETTE_MMTT=kinect
call clean.bat
call build.bat

set PALETTE_MMTT=%save_PALETTE_MMTT%

@echo off
palette stop
echo "====================== creating build with PALETTE_MMTT set to none"
set PALETTE_MMTT=none
call cleanall.bat
call build.bat
echo "====================== creating build with PALETTE_MMTT set to kinect"
set PALETTE_MMTT=kinect
call cleanall.bat
call build.bat

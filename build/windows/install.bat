echo off
set PALETTE_MMTT=none
echo "STOPPING palette"
palette stop
echo "INSTALLING binaries"
call bin_install
echo "INSTALLING data"
call data_install

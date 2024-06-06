echo off
echo "BUILDING binaries..."
call bin_build
echo "BUILDING data..."
call data_build
echo "INSTALLING binaries"
call bin_install
echo "INSTALLING data"
call data_install

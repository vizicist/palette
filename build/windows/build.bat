echo off

echo BUILDING binaries...
call build_bin

echo BUILDING data_omnisphere...
call build_data omnisphere

echo BUILDING data_sfmoma...
call build_data sfmoma

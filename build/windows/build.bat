@echo off

echo BUILDING binaries...
call build_bin

echo BUILDING data_default...
call build_data default

echo BUILDING data_dexed...
call build_data dexed


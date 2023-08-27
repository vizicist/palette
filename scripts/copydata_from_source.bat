rem This script copies things from one of the source data_* directories,
rem overwriting things in the active data directory (%CommonProgramFiles%\Palette\data).
rem This is used when working on new data (from %PALETTE_SOURCE%)
rem switching between different configurations (e.g. data_dexed vs data_omnisphere).

@echo off
if not "%1" == "" goto :doit
echo "Usage: copydata {data_dexed or data_dexedvital or data_omnisphere}"
goto getout
:doit
if exist "%PALETTE_SOURCE%\%1" goto :doit2
echo "Usage: copydata {data_dexed or data_dexedvital or data_omnisphere}"
goto getout
:doit2

rem Make a backup of and then remove saved things entirely, so that old saved things don't show up in the new configuration
rm -fr c:\tmp\palette
mkdir c:\tmp\palette
xcopy /q /e /y "%CommonProgramFiles%\Palette\data\saved" C:\tmp\palette
pushd "%CommonProgramFiles%/Palette/data/saved"
rm -fr patch quad sound
popd

xcopy /q /e /y "%PALETTE_SOURCE%\%1\config" "%CommonProgramFiles%\Palette\data\config"
xcopy /q /e /y "%PALETTE_SOURCE%\%1\saved" "%CommonProgramFiles%\Palette\data\saved"
:getout

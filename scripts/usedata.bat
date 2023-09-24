@echo off

rem This script copies things from one of the data_* directories,
rem overwriting things in the active data directory (%CommonProgramFiles%\Palette\data).
rem This is used when switching between different configurations (e.g. data_dexed vs data_omnisphere).

if not "%1" == "" goto :doit
echo "Usage: usedata {data_dexed or data_dexedvital or data_omnisphere}"
goto getout
:doit
if exist "%CommonProgramFiles%\Palette\%1" goto :doit2
echo "Usage: usedata {data_dexed or data_dexedvital or data_omnisphere}"
goto getout
:doit2

rem Make a backup of and then remove saved things entirely, so that old saved things don't show up in the new configuration
mkdir %USERPROFILE%\tmp >nul 2>nul
rm -fr %USERPROFILE%\tmp\palette
mkdir %USERPROFILE%\tmp\palette
xcopy /q /e /y "%CommonProgramFiles%\Palette\data\saved" %USERPROFILE%\tmp\palette
pushd "%CommonProgramFiles%/Palette/data/saved"
rm -fr patch quad sound
popd

pushd "%CommonProgramFiles%/Palette"
del /s _Current.json >nul 2>nul
popd

xcopy /q /e /y "%CommonProgramFiles%\Palette\%1\config" "%CommonProgramFiles%\Palette\data\config"
xcopy /q /e /y "%CommonProgramFiles%\Palette\%1\saved" "%CommonProgramFiles%\Palette\data\saved"
:getout

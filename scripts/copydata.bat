@echo off
if not "%1" == "" goto :doit
echo "Usage: copydata {data_dexed or data_dexedvital or data_omnisphere}"
goto getout
:doit
if exist "%CommonProgramFiles%\Palette\%1" goto :doit2
echo "Usage: copydata {data_dexed or data_dexedvital or data_omnisphere}"
goto getout
:doit2
@echo on
xcopy /q /e /y "%CommonProgramFiles%\Palette\data\saved" C:\tmp\palette
pushd "%CommonProgramFiles%/Palette/data/saved"
rm -fr patch quad sound
popd

xcopy /q /e /y "%CommonProgramFiles%\Palette\%1\config" "%CommonProgramFiles%\Palette\data\config"
xcopy /q /e /y "%CommonProgramFiles%\Palette\%1\saved" "%CommonProgramFiles%\Palette\data\saved"
:getout

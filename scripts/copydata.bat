@echo off
if not "%1" == "" goto :doit
echo "Usage: copydata {data_dexed or data_omnisphere}"
goto getout
:doit
@echo on
xcopy /q /e /y "%CommonProgramFiles%\Palette\%1\config" "%CommonProgramFiles%\Palette\data\config"
xcopy /q /e /y "%CommonProgramFiles%\Palette\%1\saved" "%CommonProgramFiles%\Palette\data\saved"
:getout

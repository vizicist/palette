set common="c:\program files\common files\palette\data_default\config"
echo Updating config in %common%
cd %PALETTE_SOURCE%\data_default\config
copy param*.json %common%

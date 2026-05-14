set config="%LOCALAPPDATA%\Palette\data_default\config"
echo Updating config in %config%
cd %PALETTE_SOURCE%\data_default\config
copy param*.json %config%

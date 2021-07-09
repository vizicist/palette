@echo on
call palettestopbidule.bat
set patch="%PALETTESOURCE%\default\config\palette.bidule"
echo Starting Bidule on %patch%
start /b "" "C:\Program Files\Plogue\Bidule\PlogueBidule_x64.exe" %patch%

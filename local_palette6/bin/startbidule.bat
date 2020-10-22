call killbidule.bat

set patch=%PALETTESOURCE%\local_palette6\config\palette.bidule
echo Starting Bidule on "%patch%"
start /b "" "C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_x64.exe" "%patch%"

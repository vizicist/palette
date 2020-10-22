c:/windows/system32/taskkill /F /IM palette.exe > nul 2>&1
set logdir=%LOCALAPPDATA%\Palette\logs
rm -fr "%logdir%"
mkdir "%logdir%"
echo > "%logdir%\palette.log"
echo > "%logdir%\palette.stdout"
echo > "%logdir%\palette.stderr"
start /b "" "%PALETTE%\bin\palette.exe" > "%logdir%\palette.stdout" 2> "%logdir%\palette.stderr"

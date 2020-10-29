
c:/windows/system32/taskkill /F /IM palette.exe > nul 2>&1
c:/windows/system32/taskkill /F /IM gui.exe > nul 2>&1

set logdir=%LOCALAPPDATA%\Palette\logs

echo > "%logdir%\palette.log"
echo > "%logdir%\palette.stdout"
echo > "%logdir%\palette.stderr"
start /b "" "%PALETTE%\bin\palette.exe" > "%logdir%\palette.stdout" 2> "%logdir%\palette.stderr"

rem give NATS server a chance to start
sleep 4

echo > "%logdir%\gui.log"
echo > "%logdir%\gui.stdout"
echo > "%logdir%\gui.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\gui.exe" > "%logdir%\gui.stdout" 2> "%logdir%\gui.stderr"

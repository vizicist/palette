call "%PALETTE%\bin\killgui.bat"
set logdir=%LOCALAPPDATA%\Palette\logs
mkdir "%logdir%"
echo > "%logdir%\gui.log"
echo > "%logdir%\gui.stdout"
echo > "%logdir%\gui.stderr"
start /b "" "%PALETTE%\bin\pyinstalled\gui.exe" > "%logdir%\gui.stdout" 2> "%logdir%\gui.stderr"

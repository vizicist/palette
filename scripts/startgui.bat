pushd %PALETTE%
call bin\killgui.bat
set logdir=%LOCALAPPDATA%\Palette
mkdir %logdir%
echo > %logdir%\gui.log
echo > %logdir%\gui.stdout
echo > %logdir%\gui.stderr
start /b bin\pyinstalled\gui.exe > %logdir%\gui.stdout 2> %logdir%\gui.stderr"
popd

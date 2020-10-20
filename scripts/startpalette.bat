pushd %PALETTE%
call bin\killpalette.bat
set logdir=%LOCALAPPDATA%\Palette
mkdir %logdir%
echo > %logdir%\palette.log
echo > %logdir%\palette.stdout
echo > %logdir%\palette.stderr
start /b bin\palette.exe > %logdir%\palette.stdout 2> %logdir%\palette.stderr

call %PALETTE%\bin\killpalette.bat
set logdir=%PALETTE%\logs
echo > %logdir%\palette.log
echo > %logdir%\palette.stdout
echo > %logdir%\palette.stderr
start /b %PALETTE%\bin\palette.exe > %logdir%\palette.stdout 2> %logdir%\palette.stderr

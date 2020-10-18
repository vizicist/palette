@echo off
echo running > running
:runit

python %1

if not exist runforever goto notforever
if not exist running goto getout
echo GUI has exited, will restart in 2 seconds
sleep 2
goto runit
:notforever
echo There is no runforever file, so we exit
goto finalout
:getout
echo running file was removed, that tells us to exit
:finalout

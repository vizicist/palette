:loop
time /t
echo on
call osc send 5555@127.0.0.1 /sprite 0.5 0.5 0.5 tjt
call delay 1
goto loop:

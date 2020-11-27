:loop
echo on
call osc send 3334@127.0.0.1 /sprite 0.5 0.5 0.2 tjt
timeout /t 1 > nul
goto loop:

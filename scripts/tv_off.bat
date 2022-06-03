echo standby 0 > c:/tmp/cecinput.txt
echo q >> c:/tmp/cecinput.txt
"c:\program files (x86)\pulse-eight\USB-CEC Adapter\cec-client" COM16 < c:/tmp/cecinput.txt > nul

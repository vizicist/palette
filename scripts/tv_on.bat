echo on 0 > cecinput.txt
echo q >> cecinput.txt
set com=COM7
"c:\program files (x86)\pulse-eight\USB-CEC Adapter\cec-client" %com% < cecinput.txt > nul

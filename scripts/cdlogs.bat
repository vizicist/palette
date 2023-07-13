hostname > hostname.txt
set /p hostname=<hostname.txt
del /q hostname.txt
cd %PALETTE_DATA_PATH%\logs\%hostname%

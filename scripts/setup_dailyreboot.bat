schtasks /create /sc DAILY /st 01:00 /tn "Daily Reboot for Space Palette" /tr "shutdown.exe /r /f /t 0"

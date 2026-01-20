schtasks /create /sc DAILY /st 01:00 /tn "DailyReboot" /tr "shutdown.exe /r /f /t 0"

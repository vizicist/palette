@echo off
REM Disables multi-touch gestures (three- and four-finger touch gestures) in Windows 11

REG ADD "HKCU\Control Panel\Desktop" /v TouchGestureSetting /t REG_DWORD /d 0 /f

echo Multi-touch gestures have been disabled. Please sign out and sign in again, or restart your PC for changes to take effect.
pause

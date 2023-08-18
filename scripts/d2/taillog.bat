cd "%CommonProgramFiles%\Palette\logs"
set logfile=engine
if not "%1" == "" set logfile=%1
powershell Get-Content %logfile%.log -Wait -Tail 30

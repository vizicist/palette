if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

call %PALETTESOURCE%\depthlib\clean.bat

:getout

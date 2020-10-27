if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

rm -fr %PALETTESOURCE%\ffgl\windows\x64
rm -fr %PALETTESOURCE%\windows\ship
rm -fr %PALETTESOURCE%\python\build
rm -fr %PALETTESOURCE%\python\dist

:getout

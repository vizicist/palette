if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

rm -fr %PALETTESOURCE%\ffgl\build\windows\.vs
rm -fr %PALETTESOURCE%\ffgl\build\windows\x64
rm -fr %PALETTESOURCE%\ffgl\build\windows\x86
rm -fr %PALETTESOURCE%\build\windows\ship
rm -fr %PALETTESOURCE%\python\build
rm -fr %PALETTESOURCE%\python\dist

:getout

echo ================ Compiling FFGL7 plugin
set MSBUILDCMD=C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\vsmsbuildcmd.bat
call "%MSBUILDCMD%"
pushd %PALETTESOURCE%\ffgl7\build\windows
msbuild /t:Build /p:Configuration=Debug /p:Platform="x64" palette.sln
popd

echo ================ Copying FFGL7 plugin
pushd %PALETTESOURCE%\ffgl7\windows\x64\Debug
mkdir %ship%\ffgl7
copy Palette*.dll %ship%\ffgl7 > nul
copy %PALETTESOURCE%\build\windows\pthreadvc2.dll %ship%\ffgl7 >nul
popd


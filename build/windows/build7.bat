set ship=%PALETTESOURCE%\build\windows\ship

echo ================ Compiling FFGL7 plugin
set MSBUILDCMD=C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\vsmsbuildcmd.bat
call "%MSBUILDCMD%" > nul
pushd %PALETTESOURCE%\ffgl7\build\windows
msbuild /t:Build /p:Configuration=Debug /p:Platform="x64" FFGLPlugins.sln > nul
popd
echo on

echo ================ Copying FFGL7 plugin
mkdir %ship%\ffgl7
pushd %PALETTESOURCE%\ffgl7\binaries\x64\Debug
copy Palette*.dll %ship%\ffgl7 > nul
copy %PALETTESOURCE%\build\windows\pthreadvc2.dll %ship%\ffgl7 >nul
popd

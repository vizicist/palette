
set MSBUILDCMD=C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\vsmsbuildcmd.bat
call "%MSBUILDCMD%"
pushd %PALETTESOURCE%\source\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" palette.sln
popd

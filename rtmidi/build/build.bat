msbuild /t:Build /p:Configuration=Debug /p:Platform="x64" rtmidi.vcxproj 
copy x64\Debug\rtmidi.dll ..\lib

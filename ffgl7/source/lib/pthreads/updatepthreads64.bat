@echo This must be run inside an x64 Native Tools Command Prompt for VS 2017
nmake clean VC
set bindir=..\..\..\binaries\x64
copy pthreadVC2.dll %bindir%\Debug
copy pthreadVC2.lib %bindir%\Debug
copy pthreadVC2.dll %bindir%\Release
copy pthreadVC2.lib %bindir%\Release
nmake clean
del pthreadVC2.dll
del pthreadVC2.lib


@echo off

if not "%PALETTE_SOURCE%" == "" goto keepgoing1
echo You must set the PALETTE_SOURCE environment variable.
goto getout
:keepgoing1

if not "%VSINSTALLDIR%" == "" goto keepgoing2
echo Calling msdev17 to set build environment.
call ..\..\scripts\msdev17.bat
:keepgoing2

set ship=%PALETTE_SOURCE%\build\windows\ship
set bin=%ship%\bin
rm -fr %ship% > nul 2>&1
mkdir %ship%
mkdir %ship%\bin
mkdir %ship%\ffgl
rem mkdir %ship%\bin\mmtt_kinect
rem mkdir %ship%\keykit
rem mkdir %ship%\keykit\bin
rem mkdir %ship%\keykit\lib

echo ================ Upgrading Python
python -m pip install pip | grep -v "already.*satisfied"
pip install --use-pep517 codenamize pip install python-osc requests pip install pyinstaller get-mac mido pyperclip chardet obs-cli | grep -v "already satisfied"

rem echo ================ Compiling depthlib
rem pushd ..\..\depthlib
rem call build.bat > nul
rem popd

echo ================ Creating cmds

set buildcmdsout=%PALETTE_SOURCE%\build\windows\buildcmds.out
del /f /q %buildcmdsout% >nul 2>&1

echo ================ Compiling palette
pushd %PALETTE_SOURCE%\cmd\palette
go build palette.go >> %buildcmdsout% 2>&1
move palette.exe %bin%\palette.exe > nul
popd

echo ================ Compiling palette_engine
pushd %PALETTE_SOURCE%\cmd\palette_engine
go build palette_engine.go >> %buildcmdsout% 2>&1
move palette_engine.exe %bin%\palette_engine.exe > nul
popd

echo ================ Compiling palette_monitor
pushd %PALETTE_SOURCE%\cmd\palette_monitor
go build palette_monitor.go >> %buildcmdsout% 2>&1
move palette_monitor.exe %bin%\palette_monitor.exe > nul
popd

echo ================ Compiling palette_chat
pushd %PALETTE_SOURCE%\cmd\palette_chat
go build palette_chat.go >> %buildcmdsout% 2>&1
move palette_chat.exe %bin%\palette_chat.exe > nul
popd

rem print any error messages from compiling cmds
type %buildcmdsout%

echo ================ Creating palette_gui.exe, osc.exe
pushd %PALETTE_SOURCE%\python
rm -fr dist
rm -fr build\palette_gui
rm -fr build
pyinstaller -i ..\data\config\palette.ico palette_gui.py > pyinstaller_gui.out 2>&1
pyinstaller osc.py > pyinstaller_osc.out 2>&1

echo ================ Merging python executables
rem merge all the pyinstalled things into one
move dist\palette_gui dist\pyinstalled >nul
move dist\osc\osc.exe dist\pyinstalled >nul
move dist\pyinstalled %bin% >nul
popd

echo ================ Compiling FFGL plugin
pushd %PALETTE_SOURCE%\ffgl\build\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" Palette.vcxproj > nul
popd

echo ================ Copying FFGL plugin
pushd %PALETTE_SOURCE%\ffgl\binaries\x64\Release
copy Palette*.dll %ship%\ffgl > nul
copy Palette*.pdb %ship%\ffgl > nul
copy %PALETTE_SOURCE%\build\windows\vc15\bin\pthreadvc2.dll %ship%\ffgl >nul
copy %PALETTE_SOURCE%\build\windows\vc15\bin\msvcr100.dll %ship%\ffgl >nul
popd

rem echo ================ Compiling mmtt_kinect
rem pushd %PALETTE_SOURCE%\mmtt_kinect\build\windows
rem msbuild /t:Build /p:Configuration=Debug /p:Platform="x32" mmtt_kinect.sln > nul
rem rem Put mmtt_kinect in its own bin directory, to keep 32-bit things separate
rem copy mmtt_kinect\Debug\mmtt_kinect.exe %bin%\mmtt_kinect\mmtt_kinect.exe >nul
rem copy mmtt_kinect\*.dll %bin%\mmtt_kinect >nul
rem popd

echo ================ Copying misc binaries
copy %PALETTE_SOURCE%\binaries\nircmdc.exe %bin% >nul
copy %PALETTE_SOURCE%\binaries\nats\nats.exe %bin% >nul

rem echo ================ Copying keykit things
rem copy %PALETTE_SOURCE%\keykit\bin\key.exe %ship%\keykit\bin >nul
rem copy %PALETTE_SOURCE%\keykit\bin\keylib.exe %ship%\keykit\bin >nul
rem copy %PALETTE_SOURCE%\keykit\lib\*.* %ship%\keykit\lib >nul

echo ================ Copying scripts
pushd %PALETTE_SOURCE%\scripts
copy osc.bat %bin% >nul
copy ipaddress.bat %bin% >nul
copy palette_killall.bat %bin% >nul
copy palette_onboot.bat %bin% >nul
copy delay.bat %bin% >nul
copy cdlogs.bat %bin% >nul
popd

echo ================ Copying data
mkdir %ship%\data\config
mkdir %ship%\data\midifiles
mkdir %ship%\data\saved
rem mkdir %ship%\data\keykit
rem mkdir %ship%\data\keykit\liblocal
mkdir %ship%\data\html
copy %PALETTE_SOURCE%\data\config\homepage.json %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\ffgl.json %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\param*.json %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\resolume.json %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\settings.json %ship%\data\config >nul
rem copy %PALETTE_SOURCE%\data\config\mmtt_*.json %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\synths.json %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\morphs.json %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\Palette*.avc %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\EraeTouchLayout.emk %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\*.png %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\consola.ttf %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\OpenSans-Regular.ttf %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\palette.ico %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\config\*.bidule %ship%\data\config >nul
copy %PALETTE_SOURCE%\data\midifiles\*.* %ship%\data\midifiles >nul
xcopy /e /y %PALETTE_SOURCE%\data\saved %ship%\data\saved >nul
rem xcopy /e /y %PALETTE_SOURCE%\data\keykit\liblocal %ship%\data\keykit\liblocal >nul
xcopy /e /y %PALETTE_SOURCE%\data\html %ship%\data\html >nul

echo ================ Copying windows-specific things
copy %PALETTE_SOURCE%\SenselLib\x64\LibSensel.dll %bin% >nul
copy %PALETTE_SOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin% >nul
rem copy %PALETTE_SOURCE%\depthlib\build\x64\Release\depthlib.dll %bin% >nul
copy vc15\bin\depthai-core.dll %bin% >nul
copy vc15\bin\opencv_world454.dll %bin% >nul
copy "%USERPROFILE%\mingw64\bin\libgcc_s_sjlj-1.dll" %bin% >nul
copy "%USERPROFILE%\mingw64\bin\libstdc++-6.dll" %bin% >nul

echo ================ Removing unused things
rm -fr %bin%\pyinstalled\tcl\tzdata

copy %PALETTE_SOURCE%\VERSION %ship% >nul
set /p version=<../../VERSION
echo ================ Creating installer for VERSION %version%
sed -e "s/SUBSTITUTE_VERSION_HERE/%version%/" < palette_win_setup.iss > tmp.iss
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q tmp.iss
move Output\palette_%version%_win_setup.exe %PALETTE_SOURCE%\release >nul
rmdir Output
rm tmp.iss

:getout

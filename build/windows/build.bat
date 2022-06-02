
@echo off

if not "%PALETTESOURCE%" == "" goto keepgoing1
echo You must set the PALETTESOURCE environment variable.
goto getout
:keepgoing1

if not "%VSINSTALLDIR%" == "" goto keepgoing2
echo Calling msdev17 to set build environment.
call ..\..\scripts\msdev17.bat
:keepgoing2

set ship=%PALETTESOURCE%\build\windows\ship
set bin=%ship%\bin
rm -fr %ship% > nul 2>&1
mkdir %ship%
mkdir %ship%\bin
mkdir %ship%\bin\mmtt_kinect
mkdir %ship%\config
mkdir %ship%\html
mkdir %ship%\midifiles
mkdir %ship%\ffgl

echo ================ Upgrading Python
python -m pip install pip | grep -v "already.*satisfied"
pip install codenamize pip install python-osc pip install asyncio-nats-client pyinstaller get-mac mido pyperclip | grep -v "already satisfied"

echo ================ Compiling depthlib
pushd ..\..\depthlib
call build.bat > nul
popd

echo ================ Creating palette.exe

pushd %PALETTESOURCE%\cmd\palette
go build palette.go > gobuild.out 2>&1
type nul > emptyfile
fc gobuild.out emptyfile > nul
if errorlevel 1 goto notempty
goto continue1
:notempty
echo Error in building palette.exe
cat gobuild.out
popd
goto getout
:continue1
move palette.exe %bin%\palette.exe > nul

popd

echo ================ Creating palette_engine.exe

pushd %PALETTESOURCE%\cmd\palette_engine
go build palette_engine.go > gobuild.out 2>&1
type nul > emptyfile
fc gobuild.out emptyfile > nul
if errorlevel 1 goto notempty
goto continue1
:notempty
echo Error in building palette_engine.exe
cat gobuild.out
popd
goto getout
:continue1
move palette_engine.exe %bin%\palette_engine.exe > nul

popd

echo ================ Creating palette_gui.exe, testcursor.exe, osc.exe
pushd %PALETTESOURCE%\python
rm -fr dist
rm -fr build
pyinstaller -i ..\data\config\palette.ico palette_gui.py > pyinstaller.out 2>&1
pyinstaller testcursor.py > pyinstaller.out 2>&1
pyinstaller osc.py > pyinstaller.out 2>&1

echo ================ Merging python executables
rem merge all the pyinstalled things into one
move dist\palette_gui dist\pyinstalled >nul
move dist\testcursor\testcursor.exe dist\pyinstalled >nul
move dist\osc\osc.exe dist\pyinstalled >nul
move dist\pyinstalled %bin% >nul
popd

echo ================ Compiling FFGL plugin
pushd %PALETTESOURCE%\ffgl\build\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" FFGLPlugins.sln > nul
popd

echo ================ Copying FFGL plugin
pushd %PALETTESOURCE%\ffgl\binaries\x64\Release
copy Palette*.dll %ship%\ffgl > nul
copy Palette*.pdb %ship%\ffgl > nul
copy %PALETTESOURCE%\build\windows\vc15\bin\pthreadvc2.dll %ship%\ffgl >nul
copy %PALETTESOURCE%\build\windows\vc15\bin\msvcr100.dll %ship%\ffgl >nul
popd

echo ================ Compiling mmtt_kinect
pushd %PALETTESOURCE%\mmtt_kinect\build\windows
msbuild /t:Build /p:Configuration=Debug /p:Platform="x32" mmtt_kinect.sln > nul
rem Put mmtt_kinect in its own bin directory, to keep 32-bit things separate
copy mmtt_kinect\Debug\mmtt_kinect.exe %bin%\mmtt_kinect\mmtt_kinect.exe >nul
copy mmtt_kinect\*.dll %bin%\mmtt_kinect >nul
popd

echo ================ Copying html
pushd %PALETTESOURCE%
xcopy /e /y html %ship%\html >nul
popd

echo ================ Copying misc binaries
copy %PALETTESOURCE%\binaries\nats\nats-pub.exe %bin% >nul
copy %PALETTESOURCE%\binaries\nats\nats-sub.exe %bin% >nul
copy %PALETTESOURCE%\binaries\nircmdc.exe %bin% >nul

echo ================ Copying scripts
pushd %PALETTESOURCE%\scripts
copy palettetasks.bat %bin% >nul
copy testcursor.bat %bin% >nul
copy osc.bat %bin% >nul
copy ipaddress.bat %bin% >nul
copy taillog.bat %bin% >nul
copy natsmon.bat %bin% >nul
copy delay.bat %bin% >nul
copy setpalettelogdir.bat %bin% >nul

popd

echo ================ Copying config

copy %PALETTESOURCE%\data\config\homepage.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\ffgl.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\param*.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\resolume.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\settings.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\mmtt_*.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\synths.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\morphs.json %ship%\config >nul
copy %PALETTESOURCE%\data\config\nats*.conf %ship%\config >nul
copy %PALETTESOURCE%\data\config\Palette*.avc %ship%\config >nul
copy %PALETTESOURCE%\data\config\EraeTouchLayout.emk %ship%\config >nul
copy %PALETTESOURCE%\data\config\palette.ico %ship%\config >nul
copy %PALETTESOURCE%\data\config\*.bidule %ship%\config >nul
copy %PALETTESOURCE%\data\config\attractscreen.png %ship%\config >nul
copy %PALETTESOURCE%\data\config\helpscreen.png %ship%\config >nul
copy %PALETTESOURCE%\data\config\consola.ttf %ship%\config >nul
copy %PALETTESOURCE%\data\config\OpenSans-Regular.ttf %ship%\config >nul

echo ================ Copying midifiles
copy %PALETTESOURCE%\data\midifiles\*.* %ship%\midifiles >nul

echo ================ Copying windows-specific things
copy %PALETTESOURCE%\SenselLib\x64\LibSensel.dll %bin% >nul
copy %PALETTESOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin% >nul
copy %PALETTESOURCE%\depthlib\build\x64\Release\depthlib.dll %bin% >nul
copy vc15\bin\depthai-core.dll %bin% >nul
copy vc15\bin\opencv_world454.dll %bin% >nul

echo ================ Copying presets
mkdir %ship%\presets
xcopy /e /y %PALETTESOURCE%\data\presets %ship%\presets > nul
mkdir %ship%\presets_nosuchtim
xcopy /e /y %PALETTESOURCE%\data\presets_nosuchtim %ship%\presets_nosuchtim > nul

echo ================ Removing unused things
rm -fr %bin%\pyinstalled\tcl\tzdata

copy %PALETTESOURCE%\VERSION %ship% >nul
set /p version=<../../VERSION
echo ================ Creating installer for VERSION %version%
sed -e "s/SUBSTITUTE_VERSION_HERE/%version%/" < palette_win_setup.iss > tmp.iss
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q tmp.iss
move Output\palette_%version%_win_setup.exe %PALETTESOURCE%\release >nul
rmdir Output
rm tmp.iss

:getout

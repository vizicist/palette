
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
mkdir %ship%\ffgl
mkdir %ship%\keykit
mkdir %ship%\keykit\bin
mkdir %ship%\keykit\lib

echo ================ Upgrading Python
python -m pip install pip | grep -v "already.*satisfied"
pip install --use-pep517 codenamize pip install python-osc requests pip install pyinstaller get-mac mido pyperclip chardet | grep -v "already satisfied"

rem echo ================ Compiling depthlib
rem pushd ..\..\depthlib
rem call build.bat > nul
rem popd

echo ================ Creating cmds

set buildcmdsout=%PALETTESOURCE%\build\windows\buildcmds.out
del /f /q %buildcmdsout%

echo ================ Compiling palette
pushd %PALETTESOURCE%\cmd\palette
go build palette.go >> %buildcmdsout% 2>&1
move palette.exe %bin%\palette.exe > nul
popd

echo ================ Compiling palette_engine
pushd %PALETTESOURCE%\cmd\palette_engine
go build palette_engine.go >> %buildcmdsout% 2>&1
move palette_engine.exe %bin%\palette_engine.exe > nul
popd

rem print any error messages from compiling cmds
type %buildcmdsout%

echo ================ Creating palette_gui.exe, osc.exe
pushd %PALETTESOURCE%\python
rm -fr dist
rm -fr build\palette_gui
rm -fr build
pyinstaller -i ..\data_omnisphere\config\palette.ico palette_gui.py > pyinstaller_gui.out 2>&1
pyinstaller osc.py > pyinstaller_osc.out 2>&1

echo ================ Merging python executables
rem merge all the pyinstalled things into one
move dist\palette_gui dist\pyinstalled >nul
move dist\osc\osc.exe dist\pyinstalled >nul
move dist\pyinstalled %bin% >nul
popd

echo ================ Compiling FFGL plugin
pushd %PALETTESOURCE%\ffgl\build\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" Palette.vcxproj > nul
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

echo ================ Copying misc binaries
rem copy %PALETTESOURCE%\binaries\nats\nats-pub.exe %bin% >nul
rem copy %PALETTESOURCE%\binaries\nats\nats-sub.exe %bin% >nul
copy %PALETTESOURCE%\binaries\nircmdc.exe %bin% >nul
copy %PALETTESOURCE%\binaries\tail.exe %bin% >nul

echo ================ Copying keykit things
copy %PALETTESOURCE%\keykit\bin\key.exe %ship%\keykit\bin >nul
copy %PALETTESOURCE%\keykit\bin\keylib.exe %ship%\keykit\bin >nul
copy %PALETTESOURCE%\keykit\lib\*.* %ship%\keykit\lib >nul

echo ================ Copying scripts
pushd %PALETTESOURCE%\scripts
copy palettetasks.bat %bin% >nul
copy testcursor.bat %bin% >nul
copy osc.bat %bin% >nul
copy ipaddress.bat %bin% >nul
copy taillogs.bat %bin% >nul
rem copy natsmon.bat %bin% >nul
copy delay.bat %bin% >nul
copy setpalettelogdir.bat %bin% >nul

popd

for %%X in (data_omnisphere) DO (
	echo ================ Copying %%X
	mkdir %ship%\%%X\config
	mkdir %ship%\%%X\midifiles
	mkdir %ship%\%%X\saved
	mkdir %ship%\%%X\keykit
	mkdir %ship%\%%X\keykit\liblocal
	mkdir %ship%\%%X\html
	copy %PALETTESOURCE%\%%X\config\homepage.json %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\ffgl.json %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\param*.json %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\resolume.json %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\settings.json %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\mmtt_*.json %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\synths.json %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\morphs.json %ship%\%%X\config >nul
	rem copy %PALETTESOURCE%\%%X\config\nats*.conf %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\Palette*.avc %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\EraeTouchLayout.emk %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\attractscreen.png %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\helpscreen.png %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\consola.ttf %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\OpenSans-Regular.ttf %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\palette.ico %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\config\*.bidule %ship%\%%X\config >nul
	copy %PALETTESOURCE%\%%X\midifiles\*.* %ship%\%%X\midifiles >nul
	xcopy /e /y %PALETTESOURCE%\%%X\saved %ship%\%%X\saved >nul
	xcopy /e /y %PALETTESOURCE%\%%X\keykit\liblocal %ship%\%%X\keykit\liblocal >nul
	xcopy /e /y %PALETTESOURCE%\%%X\html %ship%\%%X\html >nul
)

echo ================ Copying windows-specific things
copy %PALETTESOURCE%\SenselLib\x64\LibSensel.dll %bin% >nul
copy %PALETTESOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin% >nul
rem copy %PALETTESOURCE%\depthlib\build\x64\Release\depthlib.dll %bin% >nul
copy vc15\bin\depthai-core.dll %bin% >nul
copy vc15\bin\opencv_world454.dll %bin% >nul

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

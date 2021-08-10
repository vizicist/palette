
@echo off

if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

set ship=%PALETTESOURCE%\build\windows\ship
set bin=%ship%\bin
rm -fr %ship% > nul 2>&1
mkdir %ship%
mkdir %bin%

echo ================ Upgrading Python
python -m pip install pip | grep -v "already.*satisfied"
pip install codenamize pip install python-osc pip install asyncio-nats-client pyinstaller get-mac mido pyperclip | grep -v "already satisfied"

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

echo ================ Creating palette_gui.exe
pushd %PALETTESOURCE%\python
rm -fr dist
pyinstaller -i ..\default\config\palette.ico palette_gui.py > pyinstaller.out 2>&1
pyinstaller testcursor.py > pyinstaller.out 2>&1
pyinstaller osc.py > pyinstaller.out 2>&1

rem merge all the pyinstalled things into one
move dist\palette_gui dist\pyinstalled >nul

rem merge the other executables into that one
move dist\testcursor\testcursor.exe dist\pyinstalled >nul
move dist\osc\osc.exe dist\pyinstalled >nul
move dist\pyinstalled %bin% >nul
popd

echo ================ Compiling FFGL plugin
set MSBUILDCMD=C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\vsmsbuildcmd.bat
call "%MSBUILDCMD%" > nul
pushd %PALETTESOURCE%\ffgl\build\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" FFGLPlugins.sln > nul
popd

echo ================ Copying FFGL plugin
mkdir %ship%\ffgl
pushd %PALETTESOURCE%\ffgl\binaries\x64\Release
copy Palette*.dll %ship%\ffgl > nul
copy Palette*.pdb %ship%\ffgl > nul
copy %PALETTESOURCE%\build\windows\pthreadvc2.dll %ship%\ffgl >nul
copy %PALETTESOURCE%\build\windows\msvcr100.dll %ship%\ffgl >nul
popd

echo ================ Copying binaries
copy %PALETTESOURCE%\binaries\nats\nats-pub.exe %bin% >nul
copy %PALETTESOURCE%\binaries\nats\nats-sub.exe %bin% >nul
copy %PALETTESOURCE%\binaries\nircmdc.exe %bin% >nul

echo ================ Copying scripts
pushd %PALETTESOURCE%\scripts
copy palettestart*.bat %bin% >nul
copy palettestop*.bat %bin% >nul
copy palettetasks.bat %bin% >nul
copy testcursor.bat %bin% >nul
copy osc.bat %bin% >nul
copy ipaddress.bat %bin% >nul
copy taillog.bat %bin% >nul
copy natsmon.bat %bin% >nul
copy delay.bat %bin% >nul
copy resizegui.bat %bin% >nul
copy setpalettelogdir.bat %bin% >nul

popd

echo ================ Copying config
mkdir %ship%\config

copy %PALETTESOURCE%\default\config\ffgl.json %ship%\config >nul
copy %PALETTESOURCE%\default\config\param*.json %ship%\config >nul
copy %PALETTESOURCE%\default\config\resolume.json %ship%\config >nul
copy %PALETTESOURCE%\default\config\settings.json %ship%\config >nul
copy %PALETTESOURCE%\default\config\synths.json %ship%\config >nul
copy %PALETTESOURCE%\default\config\morphs.json %ship%\config >nul
copy %PALETTESOURCE%\default\config\nats*.conf %ship%\config >nul
copy %PALETTESOURCE%\default\config\Palette*.avc %ship%\config >nul
copy %PALETTESOURCE%\default\config\palette.ico %ship%\config >nul
copy %PALETTESOURCE%\default\config\palette.bidule %ship%\config >nul

echo ================ Copying midifiles
mkdir %ship%\midifiles
copy %PALETTESOURCE%\default\midifiles\*.* %ship%\midifiles >nul

echo ================ Copying windows-specific things
copy %PALETTESOURCE%\SenselLib\x64\LibSensel.dll %bin% >nul
copy %PALETTESOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin% >nul

echo ================ Copying presets
mkdir %ship%\presets
xcopy /e /y %PALETTESOURCE%\default\presets %ship%\presets > nul

echo ================ Removing unused things
rm -fr %bin%\pyinstalled\tcl\tzdata

echo ================ Creating installer
"c:\Program Files (x86)\Inno Setup 6\ISCC.exe" /Q palette_win_setup.iss
if exist "T:\\tmp" (
	copy Output\palette_*_win_setup.exe t:\tmp >nul
)
move Output\palette_*_win_setup.exe %PALETTESOURCE%\release >nul
rmdir Output

:getout

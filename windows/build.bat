
@echo off

if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

set ship=%PALETTESOURCE%\windows\ship
set bin=%ship%\bin
rm -fr %ship% > nul 2>&1
mkdir %ship%
mkdir %bin%

echo ================ NOT Upgrading Python - takes too long
rem python -m pip install --upgrade pip
rem pip install codenamize pip install python-osc pip install asyncio-nats-client pyinstaller get-mac | grep -v "already satisfied"

echo ================ Compiling FFGL plugin
set MSBUILDCMD=C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\vsmsbuildcmd.bat
call "%MSBUILDCMD%" > nul
pushd %PALETTESOURCE%\ffgl\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" palette.sln > nul
msbuild /t:Build /p:Configuration=Debug /p:Platform="x64" palette.sln > nul
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

echo ================ Creating palette_gui_*.exe
pushd %PALETTESOURCE%\python
rm -fr dist
pyinstaller palette_gui_full.py > pyinstaller.out 2>&1
pyinstaller palette_gui_remote.py > pyinstaller.out 2>&1
pyinstaller testcursor.py > pyinstaller.out 2>&1
pyinstaller osc.py > pyinstaller.out 2>&1

rem merge all the pyinstalled things into one
move dist\palette_gui_full dist\pyinstalled >nul

rem merge the other executables into that one
move dist\palette_gui_remote\palette_gui_remote.exe dist\pyinstalled >nul
move dist\testcursor\testcursor.exe dist\pyinstalled >nul
move dist\osc\osc.exe dist\pyinstalled >nul
move dist\pyinstalled %bin% >nul
popd

echo ================ Copying FFGL PLUGIN
pushd %PALETTESOURCE%\ffgl\windows\x64\Debug
mkdir %ship%\ffgl
copy *.* %ship%\ffgl > nul
popd

echo ================ Copying NATS
copy %PALETTESOURCE%\nats\nats-pub.exe %bin% >nul
copy %PALETTESOURCE%\nats\nats-sub.exe %bin% >nul

echo ================ Copying scripts
pushd %PALETTESOURCE%\scripts
copy palettestart*.bat %bin% >nul
copy palettestop*.bat %bin% >nul
copy testcursor.bat %bin% >nul
copy osc.bat %bin% >nul
copy taillog.bat %bin% >nul
copy natsmon.bat %bin% >nul

popd

echo ================ Copying config
mkdir %ship%\config
copy %PALETTESOURCE%\default\config\*.json %ship%\config >nul
copy %PALETTESOURCE%\default\config\*.conf %ship%\config >nul

echo ================ Copying midifiles
mkdir %ship%\midifiles
copy %PALETTESOURCE%\default\midifiles\*.* %ship%\midifiles >nul

echo ================ Copying windows-specific things
copy %PALETTESOURCE%\windows\pthreadvc2.dll %ship%\ffgl >nul
copy %PALETTESOURCE%\windows\msvcp140d.dll %ship%\ffgl >nul
copy %PALETTESOURCE%\SenselLib\x64\LibSensel.dll %bin% >nul
copy %PALETTESOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin% >nul

echo ================ Copying presets
mkdir %ship%\presets
xcopy /e /y %PALETTESOURCE%\default\presets %ship%\presets > nul

echo ================ Removing unused things
rm -fr %bin%\pyinstalled\tcl\tzdata

call buildsetup.bat

:getout

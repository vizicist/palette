
@echo off

if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

echo Calling killall
call %PALETTESOURCE%\scripts\killall.bat

set ship=%PALETTESOURCE%\windows\ship
set bin=%ship%\bin
rm -fr %ship% > nul 2>&1
mkdir %ship%
mkdir %bin%

echo ================ NOT Upgrading Python
rem python -m pip install --upgrade pip
rem pip install codenamize pip install python-osc pip install asyncio-nats-client pyinstaller get-mac | grep -v "already satisfied"

echo ================ COMPILING FFGL PLUGIN
set MSBUILDCMD=C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\vsmsbuildcmd.bat
call "%MSBUILDCMD%"
pushd %PALETTESOURCE%\ffgl\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" palette.sln
msbuild /t:Build /p:Configuration=Debug /p:Platform="x64" palette.sln
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
move palette.exe %bin%\palette.exe

popd

echo ================ Creating gui.exe
pushd %PALETTESOURCE%\python
rm -fr dist
pyinstaller gui.py > pyinstaller.out 2>&1
pyinstaller testcursor.py > pyinstaller.out 2>&1
pyinstaller osc.py > pyinstaller.out 2>&1
rem merge them all into one
move dist\gui dist\pyinstalled
mv dist\testcursor\testcursor.exe dist\pyinstalled
mv dist\osc\osc.exe dist\pyinstalled
move dist\pyinstalled %bin%
popd

echo ================ COPYING FFGL PLUGIN
pushd %PALETTESOURCE%\ffgl\windows\x64\Debug
mkdir %ship%\ffgl
copy *.* %ship%\ffgl
popd

echo ================ COPYING nats
copy %PALETTESOURCE%\nats\nats-pub.exe %bin%
copy %PALETTESOURCE%\nats\nats-sub.exe %bin%

echo ================ COPYING scripts
pushd %PALETTESOURCE%\scripts
copy killall.bat %bin%
copy killpalette.bat %bin%
copy killgui.bat %bin%
copy testcursor.bat %bin%
copy osc.bat %bin%
copy taillog.bat %bin%

copy startall.bat %bin%
copy startpalette.bat %bin%
copy startgui.bat %bin%

copy natsmon.bat %bin%

popd

echo ================ COPYING config
mkdir %ship%\config
copy %PALETTESOURCE%\default\config\*.json %ship%\config
copy %PALETTESOURCE%\default\config\*.conf %ship%\config

echo ================ COPYING windows-specific things
copy %PALETTESOURCE%\windows\pthreadvc2.dll %ship%\ffgl
copy %PALETTESOURCE%\windows\msvcp140d.dll %ship%\ffgl
copy %PALETTESOURCE%\SenselLib\x64\LibSensel.dll %bin%
copy %PALETTESOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin%

echo ================ COPYING presets
mkdir %ship%\presets
xcopy /e /y %PALETTESOURCE%\default\presets %ship%\presets > nul

echo ================ REMOVING UNUSED THINGS
rm -fr %bin%\pyinstalled\tcl\tzdata
rm -fr %bin%\pyinstalled\tcl\encoding

call buildsetup.bat

:getout

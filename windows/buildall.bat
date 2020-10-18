
@echo off

if not "%PALETTESOURCE%" == "" goto keepgoing
echo You must set the PALETTESOURCE environment variable.
goto getout

:keepgoing

echo Calling killall
call %PALETTESOURCE%\scripts\killall.bat

set SHIPNAME=palette_win

set ship=%PALETTESOURCE%\ship\%SHIPNAME%
rm -fr %ship% > nul 2>&1
mkdir %ship%

echo ================ Upgrading Python
python -m pip install --upgrade pip
pip install codenamize pip install python-osc pip install asyncio-nats-client pyinstaller get-mac | grep -v "already satisfied"

echo ================ COMPILING FFGL PLUGIN
set MSBUILDCMD=C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\Tools\vsmsbuildcmd.bat
call "%MSBUILDCMD%"
pushd %PALETTESOURCE%\source\windows
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" palette.sln
msbuild /t:Build /p:Configuration=Debug /p:Platform="x64" palette.sln
popd

rem The popd for this is at the end of the file
pushd %PALETTESOURCE%\cmd\palette

echo ================ Creating palette.exe

set bin=%ship%\bin
mkdir %bin%

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
move palette.exe %bin%
popd

echo ================ Creating gui.exe
pushd %PALETTESOURCE%\python
rm -fr dist
pyinstaller gui.py > pyinstaller.out 2>&1
pyinstaller testcursor.py > pyinstaller.out 2>&1
pyinstaller oscsend.py > pyinstaller.out 2>&1
pyinstaller osclisten.py > pyinstaller.out 2>&1
pyinstaller startbidule.py > pyinstaller.out 2>&1
pyinstaller startresolume.py > pyinstaller.out 2>&1
rem merge them all into one
move dist\gui dist\pyinstalled
mv dist\testcursor\testcursor.exe dist\pyinstalled
mv dist\oscsend\oscsend.exe dist\pyinstalled
mv dist\osclisten\osclisten.exe dist\pyinstalled
mv dist\startresolume\startresolume.exe dist\pyinstalled
mv dist\startbidule\startbidule.exe dist\pyinstalled
move dist\pyinstalled %bin%
popd

echo ================ COPYING FFGL PLUGIN
pushd %PALETTESOURCE%\source\windows\x64\Debug
mkdir %ship%\ffgl
copy *.* %ship%\ffgl
popd

echo ================ COPYING nats
mkdir %ship%\nats
copy %PALETTESOURCE%\nats\*.* %ship%\nats

echo ================ COPYING scripts
pushd %PALETTESOURCE%\scripts
copy killall.bat %bin%
copy killpalette.bat %bin%
copy killgui.bat %bin%
copy killresolume.bat %bin%
copy killbidule.bat %bin%
copy testcursor.bat %bin%

copy startall.bat %bin%
copy startpalette.bat %bin%
copy startgui.bat %bin%
copy startresolume.bat %bin%
copy startbidule.bat %bin%

copy natsmon.bat %bin%
popd

echo ================ COPYING config
mkdir %ship%\config
mkdir %ship%\logs > nul 2>&1

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

echo ================ CREATING %SHIPNAME%.zip
pushd %PALETTESOURCE%\ship
rm -f %SHIPNAME%.zip
powershell Compress-Archive -Path %SHIPNAME% -CompressionLevel Fastest -DestinationPath %SHIPNAME%.zip

popd

:getout

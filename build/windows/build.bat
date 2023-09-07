
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
rm -fr %ship%
mkdir %ship%
mkdir %ship%\bin
mkdir %ship%\bin\mmtt_kinect
mkdir %ship%\ffgl

echo ================ Upgrading Python
python -m pip install pip | grep -v "already.*satisfied"
pip install --use-pep517 codenamize pip install python-osc requests pip install pyinstaller get-mac mido pyperclip chardet | grep -v "already satisfied"

rem echo ================ Compiling depthlib
rem pushd ..\..\depthlib
rem call build.bat > nul
rem popd

echo ================ Creating cmds

set buildcmdsout=%PALETTE_SOURCE%\build\windows\buildcmds.out
del /f /q %buildcmdsout% >nul 2>&1

echo ================ Compiling palette
pushd %PALETTE_SOURCE%\cmd\palette
go build -o palette.exe >> %buildcmdsout% 2>&1
move palette.exe %bin%\palette.exe > nul
popd

echo ================ Compiling palette_engine
pushd %PALETTE_SOURCE%\cmd\palette_engine
go build -o palette_engine.exe >> %buildcmdsout% 2>&1
move palette_engine.exe %bin%\palette_engine.exe > nul
popd

echo ================ Compiling palette_monitor
pushd %PALETTE_SOURCE%\cmd\palette_monitor
go build -o palette_monitor.exe >> %buildcmdsout% 2>&1
move palette_monitor.exe %bin%\palette_monitor.exe > nul
popd

rem echo ================ Compiling palette_splash
rem pushd %PALETTE_SOURCE%\cmd\palette_splash
rem go build -o palette_splash.exe >> %buildcmdsout% 2>&1
rem move palette_splash.exe %bin%\palette_splash.exe > nul
rem popd

rem echo ================ Compiling palette_pk2go
rem pushd %PALETTE_SOURCE%\cmd\palette_pk2go
rem go build -o palette_pk2go.exe >> %buildcmdsout% 2>&1
rem move palette_pk2go.exe %bin%\palette_pk2go.exe > nul
rem popd

rem print any error messages from compiling cmds
type %buildcmdsout%

echo ================ Creating palette_gui.exe, osc.exe
pushd %PALETTE_SOURCE%\python
rm -fr dist
rm -fr build\palette_gui
rm -fr build
pyinstaller -i ..\data_defaults\config\palette.ico palette_gui.py > pyinstaller_gui.out 2>&1
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
rem copy %PALETTE_SOURCE%\binaries\nats\nats-pub.exe %bin% >nul
rem copy %PALETTE_SOURCE%\binaries\nats\nats-sub.exe %bin% >nul
copy %PALETTE_SOURCE%\binaries\nircmdc.exe %bin% >nul

echo ================ Copying scripts
pushd %PALETTE_SOURCE%\scripts
copy palettetasks.bat %bin% >nul
copy testcursor.bat %bin% >nul
copy osc.bat %bin% >nul
copy ipaddress.bat %bin% >nul
copy taillog.bat %bin% >nul
copy taillogs.bat %bin% >nul
copy palette_killall.bat %bin% >nul
copy palette_restart.bat %bin% >nul
copy palette_onboot.bat %bin% >nul
rem copy natsmon.bat %bin% >nul
copy delay.bat %bin% >nul
copy setpalettelogdir.bat %bin% >nul
copy cdlogs.bat %bin% >nul

popd

mkdir %ship%\data\config
mkdir %ship%\data\html
mkdir %ship%\data\saved
xcopy /e /y %PALETTE_SOURCE%\data_defaults\config %ship%\data\config >nul
xcopy /e /y %PALETTE_SOURCE%\data_defaults\html %ship%\data\html >nul
xcopy /e /y %PALETTE_SOURCE%\data_defaults\saved %ship%\data\saved >nul

rem The default data directory is data_dexedvital
rem You can use copydata.bat to switch to other data_*.
xcopy /e /y %PALETTE_SOURCE%\data_dexedvital\saved %ship%\data\saved >nul
xcopy /e /y %PALETTE_SOURCE%\data_dexedvital\config %ship%\data\config >nul

for %%X in (data_dexed data_dexedvital data_omnisphere) DO (
	echo ================ Copying %%X
	mkdir %ship%\%%X\config
	mkdir %ship%\%%X\saved
	xcopy /e /y %PALETTE_SOURCE%\%%X\config %ship%\%%X\config >nul
	xcopy /e /y %PALETTE_SOURCE%\%%X\saved %ship%\%%X\saved >nul
)

echo ================ Copying windows-specific things
copy %PALETTE_SOURCE%\SenselLib\x64\LibSensel.dll %bin% >nul
copy %PALETTE_SOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin% >nul
rem copy %PALETTE_SOURCE%\depthlib\build\x64\Release\depthlib.dll %bin% >nul
rem copy vc15\bin\depthai-core.dll %bin% >nul
rem copy vc15\bin\opencv_world454.dll %bin% >nul

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

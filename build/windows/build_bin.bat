
@echo off

if not "%PALETTE_SOURCE%" == "" goto keepgoing1
	echo You must set the PALETTE_SOURCE environment variable.
	goto getout
:keepgoing1

if not "%VSINSTALLDIR%" == "" goto keepgoing2
	call :set_msdev_env
	if errorlevel 1 goto getout
:keepgoing2

call :check_mingw64
if errorlevel 1 goto getout

set ship=%PALETTE_SOURCE%\build\windows\ship
set bin=%ship%\bin
rm -fr %ship% > nul 2>&1
mkdir %ship%
mkdir %ship%\bin
mkdir %ship%\ffgl
rem mkdir %ship%\keykit
rem mkdir %ship%\keykit\bin
rem mkdir %ship%\keykit\lib

copy %PALETTE_SOURCE%\VERSION %ship% >nul
set /p version=<../../VERSION
rem putting the PALETTE_VERSION in the environment so it can be used in the installer
set PALETTE_VERSION=%version%

echo ================ Upgrading Python
python -m pip install pip | grep -v "already.*satisfied"
pip install --use-pep517 pip python-osc requests pip pyinstaller get-mac mido pyperclip chardet obs-cli | grep -v "already satisfied"

rem echo ================ Compiling depthlib
rem pushd ..\..\depthlib
rem call build.bat > nul
rem popd

if "%PALETTE_MMTT%" == "" goto no_mmtt
	echo ================ Building mmtt
	call build_mmtt.bat > build_mmtt.out 2>&1
:no_mmtt

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

echo ================ Compiling palette_hub
pushd %PALETTE_SOURCE%\cmd\palette_hub
go build palette_hub.go >> %buildcmdsout% 2>&1
move palette_hub.exe %bin%\palette_hub.exe > nul
popd

rem print any error messages from compiling cmds
type %buildcmdsout%

echo ================ Compiling FFGL plugin
pushd %PALETTE_SOURCE%\ffgl\build\windows
set PALETTE_DATA=default
msbuild /t:Build /p:Configuration=Release /p:Platform="x64" Palette.vcxproj >> %buildcmdsout% 2>&1
if errorlevel 1 (
	type %buildcmdsout%
	popd
	goto getout
)
popd

echo ================ Copying FFGL plugin
if not exist "%PALETTE_SOURCE%\ffgl\binaries\x64\Release\Palette*.dll" (
	echo The FFGL plugin build did not create Palette*.dll under:
	echo     %PALETTE_SOURCE%\ffgl\binaries\x64\Release
	goto getout
)
pushd %PALETTE_SOURCE%\ffgl\binaries\x64\Release
copy Palette*.dll %ship%\ffgl > nul
copy Palette*.pdb %ship%\ffgl > nul
copy %PALETTE_SOURCE%\build\windows\vc15\bin\pthreadvc2.dll %ship%\ffgl >nul
copy %PALETTE_SOURCE%\build\windows\vc15\bin\msvcr100.dll %ship%\ffgl >nul
popd

rem ======== Kinect (mmtt_kinect) is only built when PALETTE_MMTT is set
rem ======== NOTE - this is now NOT done automatically, you need to call build_mmtt manually
rem if "%PALETTE_MMTT%" == "kinect" call build_mmtt.bat

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
copy palette_onboot_kinect.bat %bin% >nul
copy delay.bat %bin% >nul
copy cdlogs.bat %bin% >nul
copy cddata.bat %bin% >nul
copy cdglobal.bat %bin% >nul
copy cdconfig.bat %bin% >nul
copy tv_on.bat %bin% >nul
copy tv_off.bat %bin% >nul
copy disable_multitouch.bat %bin% >nul
copy setup_onboot.bat %bin% >nul
copy remove_onboot.bat %bin% >nul
copy setup_dailyreboot.bat %bin% >nul
copy remove_dailyreboot.bat %bin% >nul
popd

echo ================ Copying windows-specific things
copy %PALETTE_SOURCE%\SenselLib\x64\LibSensel.dll %bin% >nul
copy %PALETTE_SOURCE%\SenselLib\x64\LibSenselDecompress.dll %bin% >nul
rem copy %PALETTE_SOURCE%\depthlib\build\x64\Release\depthlib.dll %bin% >nul
copy vc15\bin\depthai-core.dll %bin% >nul
copy vc15\bin\opencv_world454.dll %bin% >nul
copy "%USERPROFILE%\mingw64\bin\libwinpthread-1.dll" %bin% >nul
copy "%USERPROFILE%\mingw64\bin\libgcc_s_seh-1.dll" %bin% >nul
if exist "%USERPROFILE%\mingw64\bin\libgcc_s_sjlj-1.dll" copy "%USERPROFILE%\mingw64\bin\libgcc_s_sjlj-1.dll" %bin% >nul
copy "%USERPROFILE%\mingw64\bin\libstdc++-6.dll" %bin% >nul

echo ================ Removing unused things
if exist %bin%\pyinstalled\tcl\tzdata rm -fr %bin%\pyinstalled\tcl\tzdata

echo ================ Creating installer for VERSION %version%

call :find_inno_setup
if errorlevel 1 goto getout

"%ISCC_EXE%" /Q palette_win_setup.iss

if not "%PALETTE_MMTT%" == "kinect" goto no_kinect
move Output\palette_%version%_win_setup.exe %PALETTE_SOURCE%\release\palette_%version%_win_setup_with_kinect.exe >nul
goto finish

:no_kinect
move Output\palette_%version%_win_setup.exe %PALETTE_SOURCE%\release >nul

:finish

rmdir Output

:getout
set PALETTE_VERSION=
goto :eof

:set_msdev_env
set "vswhere=%ProgramFiles(x86)%\Microsoft Visual Studio\Installer\vswhere.exe"
set "vcvars64="

if not exist "%vswhere%" goto try_msdev_scripts

for /f "usebackq delims=" %%i in (`"%vswhere%" -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath`) do set "vcvars64=%%i\VC\Auxiliary\Build\vcvars64.bat"

if "%vcvars64%" == "" goto try_msdev_scripts
if not exist "%vcvars64%" goto try_msdev_scripts

echo Calling "%vcvars64%" to set build environment.
call "%vcvars64%"
exit /b %ERRORLEVEL%

:try_msdev_scripts
for /f "delims=" %%i in ('dir /b /o-n "%~dp0..\..\scripts\msdev*.bat" 2^>nul') do call :try_msdev_script "%%i" && exit /b 0

echo Unable to find a Visual Studio build environment.
echo Install Visual Studio with C++ build tools, or add an msdev*.bat script under %PALETTE_SOURCE%\scripts.
exit /b 1

:try_msdev_script
echo Calling %~1 to set build environment.
call "%~dp0..\..\scripts\%~1"
if not "%VSINSTALLDIR%" == "" exit /b 0
exit /b 1

:check_mingw64
set "MINGW64_BIN=%USERPROFILE%\mingw64\bin"

if not exist "%MINGW64_BIN%" goto missing_mingw64
if not exist "%MINGW64_BIN%\gcc.exe" goto missing_mingw64
if not exist "%MINGW64_BIN%\g++.exe" goto missing_mingw64

set "PATH=%MINGW64_BIN%;%PATH%"
set CGO_ENABLED=1
exit /b 0

:missing_mingw64
echo MinGW-w64 is required for the Windows build, but it was not found.
echo Expected gcc.exe and g++.exe under:
echo     %MINGW64_BIN%
echo Install MinGW-w64 there, or update build\windows\build_bin.bat if your install lives somewhere else.
exit /b 1

:find_inno_setup
set "ISCC_EXE="

for %%i in (ISCC.exe) do if not "%%~$PATH:i" == "" set "ISCC_EXE=%%~$PATH:i"

if not "%ISCC_EXE%" == "" exit /b 0
if exist "%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe" set "ISCC_EXE=%ProgramFiles(x86)%\Inno Setup 6\ISCC.exe"
if exist "%ProgramFiles%\Inno Setup 6\ISCC.exe" set "ISCC_EXE=%ProgramFiles%\Inno Setup 6\ISCC.exe"

if not "%ISCC_EXE%" == "" exit /b 0

echo Inno Setup 6 is required to create the Windows installer, but ISCC.exe was not found.
echo Install Inno Setup 6, or add ISCC.exe to PATH.
exit /b 1

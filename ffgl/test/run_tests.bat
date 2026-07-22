@echo off
rem Builds and runs the standalone FFGL unit tests. These cover code that has
rem no dependency on OpenGL or the FFGL SDK, so they compile with plain g++
rem (the same MinGW-w64 the Go build already requires) and run in a second.

setlocal

if not "%PALETTE_SOURCE%" == "" goto keepgoing
	echo You must set the PALETTE_SOURCE environment variable.
	exit /b 1
:keepgoing

set "GXX=%USERPROFILE%\mingw64\bin\g++.exe"
if not exist "%GXX%" (
	echo Could not find g++ at %GXX%
	echo Install MinGW-w64 there, the same way build\windows\build_bin.bat expects it.
	exit /b 1
)

set "src=%PALETTE_SOURCE%\ffgl\source\lib\palette"
set "out=%TEMP%\palette_ffgl_tests"
if not exist "%out%" mkdir "%out%"

echo ================ test_polygonfill
"%GXX%" -std=c++14 -O2 -Wall ^
	-I "%src%" -I "%PALETTE_SOURCE%\ffgl\source\lib\glm" ^
	-o "%out%\test_polygonfill.exe" ^
	"%PALETTE_SOURCE%\ffgl\test\test_polygonfill.cpp" ^
	"%src%\PolygonFill.cpp"
if errorlevel 1 exit /b 1
"%out%\test_polygonfill.exe"
if errorlevel 1 exit /b 1

echo.
echo All FFGL tests passed.
exit /b 0

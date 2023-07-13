set xx=%1
echo ================ Compiling %xx%

echo on
pushd %PALETTE_SOURCE%\cmd\%xx%
go build %xx%.go > gobuild.out 2>&1

for /f %%i in ("gobuild.out") do (
	set size=%%~zi
	echo "size is %size%"
)
echo size again is "%size%"
if "%size%" gtr "0" goto notempty
goto continue1
:notempty
echo Error in building %xx%
cat gobuild.out
popd
goto getout
:continue1
echo NO ERROR in building %xx%

:getout

	echo ================ Compiling mmtt_kinect
	pushd %PALETTE_SOURCE%\mmtt_kinect\build\windows
	msbuild /t:Build /p:Configuration=Debug /p:Platform="x32" mmtt_kinect.sln
	rem Put mmtt_kinect in its own bin directory, to keep 32-bit things separate
	mkdir %bin%\mmtt_kinect
	copy mmtt_kinect\Debug\mmtt_kinect.exe %bin%\mmtt_kinect\mmtt_kinect.exe >nul
	copy mmtt_kinect\*.dll %bin%\mmtt_kinect >nul
	popd

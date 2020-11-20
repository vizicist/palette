pushd ..\..\binaries\x64\Debug

echo Copying Palette.dll to Palette_[1234].dll
copy Palette.dll Palette_1.dll
copy Palette.dll Palette_2.dll
copy Palette.dll Palette_3.dll
copy Palette.dll Palette_4.dll

popd

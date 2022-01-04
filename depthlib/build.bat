pushd build
msbuild /target:depthlib /p:Configuration=Debug /p:Platform="x64" depthlib.sln
popd

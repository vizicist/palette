pushd build
msbuild /target:depthlib /p:Configuration=Release /p:Platform="x64" depthlib.sln
popd

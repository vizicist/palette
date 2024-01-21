@echo off
echo "=========== Making libspout.a"
mingw32-make.exe clean
mingw32-make.exe default
go build
go install

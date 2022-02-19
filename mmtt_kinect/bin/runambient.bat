c:/windows/system32/taskkill /f /im loopMIDI.exe
c:/windows/system32/taskkill /f /im PlogueBidule_x64.exe
c:/windows/system32/taskkill /f /im Avenue.exe
c:/windows/system32/taskkill /f /im Arena.exe
sleep 1

cd \Program Files (x86)\Tobias Erichsen\loopMIDI
sh -c "./loopMIDI.exe &"
sleep 4

start c:/local/manifold/patches/bidule/Palette_Alchemy_Ambient.bidule
sleep 25
sleep 20

set patches=c:\local\manifold\src\manifold_bm\Binaries\win32\Manifold\patches
copy %patches%\default_ambient.mnf %patches%\default.mnf

set patches=c:\local\manifold\src\ffglplugins\Binaries\win32\Manifold\patches
copy %patches%\default_ambient.mnf %patches%\default.mnf

call cdarena.bat
start Arena.exe < nul > nul 2> nul

rem call cdarena.bat
rem start Avenue.exe < nul > nul 2> nul

sleep 5
cd \local\manifold\src\oscutil
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer2/clip1/connect 1
c:\python26\python.exe oscsend.py 7000@127.0.0.1 /layer1/clip1/connect 1
sleep 2

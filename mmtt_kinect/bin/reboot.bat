call c:/python27/python.exe c:/local/manifold/bin/killall.py
start c:/python27/python.exe c:/local/manifold/bin/splash.py "Sorry, the brushes need cleaning." "A restart is underway." "Back in a few minutes!"
sleep 10
c:/windows/system32/shutdown.exe -t 0 -r -f

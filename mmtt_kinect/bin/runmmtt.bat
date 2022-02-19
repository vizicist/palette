c:/windows/system32/taskkill /f /im mmtt.exe
c:/windows/system32/taskkill /f /im mmtt_depth.exe
rem set PUBLIC=c:\local\manifold\Public
if x%1 == x start mmtt_depth.exe -r -cdefault
if not x%1 == x start mmtt_depth.exe -r -c%1

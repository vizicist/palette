import sys

import palette

if len(sys.argv) != 6:
    print("Usage: cursorevent {cid} {down|drag|up} {x} {y} {z}")
    sys.exit(1)

cid = sys.argv[1]
ddu = sys.argv[2]
if ddu != "down" and ddu != "drag" and ddu != "up":
    print("Bad value, expecting down/drag/up as first argument")
    sys.exit(1)
x = float(sys.argv[3])
y = float(sys.argv[4])
z = float(sys.argv[5])

palette.SendCursorEvent(cid,ddu,x,y,z)

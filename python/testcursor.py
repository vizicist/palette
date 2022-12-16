import sys
import time
import random
import palette

if len(sys.argv) > 1:
        ntimes = int(sys.argv[1])
else:
        ntimes = 10

if len(sys.argv) > 2:
        dt = float(sys.argv[2])
else:
        dt = 0.1

cid = "0"
for n in range(ntimes):
        # Random layer
        layer = "ABCD"[random.randint(0,3)]
        palette.SendCursorEvent(cid,"down",random.random(),random.random(),random.random()/4.0,layer=layer)
        time.sleep(dt)
        palette.SendCursorEvent(cid,"up",random.random(),random.random(),0.0,layer=layer)
        time.sleep(dt)

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
palette.FakeNUID(palette.MyNUID()+"_test")
for n in range(ntimes):
        palette.SendCursorEvent(cid,"down",random.random(),random.random(),random.random()/4.0)
        time.sleep(dt)
        palette.SendCursorEvent(cid,"up",random.random(),random.random(),0.0)
        time.sleep(dt)

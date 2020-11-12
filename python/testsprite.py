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
    x = random.random()
    y = random.random()
    z = random.random() / 4.0
    palette.SendSpriteEvent(cid,x,y,z)
    time.sleep(dt)

import sys
import time
import random
import palette

dt = 0.1
cid = "0"

layer = "A"
palette.SendSpriteEvent(cid,0.1,0.1,0.75,layer=layer)
time.sleep(dt)
palette.SendSpriteEvent(cid,0.9,0.9,0.75,layer=layer)
time.sleep(dt)

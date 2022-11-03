import sys
import time
import random
import palette

dt = 0.1
cid = "0"
player = "A"

palette.SendCursorEvent(cid,"down",0.1,0.1,0.5,player=player)
time.sleep(dt)
palette.SendCursorEvent(cid,"up",0.1,0.1,0.0,player=player)
time.sleep(dt)

palette.SendCursorEvent(cid,"down",0.9,0.9,0.5,player=player)
time.sleep(dt)
palette.SendCursorEvent(cid,"up",0.9,0.9,0.0,player=player)
time.sleep(dt)

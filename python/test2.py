import sys
import time
import random
import palette

dt = 0.1
cid = "0"

player = "A"
palette.SendSpriteEvent(cid,0.1,0.1,0.75,player=player)
time.sleep(dt)
palette.SendSpriteEvent(cid,0.9,0.9,0.75,player=player)
time.sleep(dt)

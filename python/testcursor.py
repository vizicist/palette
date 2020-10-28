import glob
import os
import sys
import time
import traceback
import json
import collections
import random
from subprocess import call, Popen

from pythonosc import udp_client
from urllib import parse, request

import spaceutil

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
        spaceutil.SendCursorEvent(cid,"down",x,y,z)
        time.sleep(dt)
        x = random.random()
        y = random.random()
        spaceutil.SendCursorEvent(cid,"up",x,y,0.0)
        time.sleep(dt)

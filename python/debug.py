import glob
import os
import sys
import time
import traceback
import json
import collections
from subprocess import call, Popen

from pythonosc import udp_client
from urllib import parse, request

import palette

if len(sys.argv) != 3:
    print("Usage: debug {debug-type} {onoff}")
    sys.exit(1)

dtype = sys.argv[1]
onoff = sys.argv[2]
palette.palette_api("global.debug","{ \"debug\": \""+dtype+"\", \"onoff\": \""+onoff+"\" }" )

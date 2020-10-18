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

import spaceutil

r = spaceutil.palette_api("global.echo","{ \"value\": \"plugh\" }" )
print("r=",r)

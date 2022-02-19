# reset audio of Space Palette
#
import json
from subprocess import call, Popen
from time import sleep
from traceback import format_exc
from threading import Thread, Lock

from nosuch.oscutil import *

global Plogue
plogue = OscRecipient("127.0.0.1",3210)

sleep(2)
print("sending off")
plogue.sendosc("/play",[0])
sleep(2)
print("sending on")
plogue.sendosc("/play",[1])

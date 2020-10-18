import sys
import os
import json
import signal
import socket

from time import sleep
from subprocess import call, Popen
from pythonosc import udp_client

import spaceutil

palettedir = os.getenv("PALETTE")

palettedirbackslash = palettedir.replace("/","\\")

# Add location of the plugin binaries to PATH so Resolume will find them
ffglpath = os.path.join(palettedirbackslash,"ffgl")
print("Adding FFGL directory to PATH: "+ffglpath)
os.environ["PATH"] = ffglpath + ";" + os.environ["PATH"]

# f = open(os.path.join(palettedirbackslash,"config","settings.json"))
# j = json.load(f)
# f.close()

resolumelayers = spaceutil.ConfigValue("resolumelayers")
resolumedelay = spaceutil.ConfigValue("resolumedelay")

print("Starting Resolume...")

dummy = Popen(["C:\\Program Files\\Resolume Avenue 6\\Avenue.exe"])

resolume = udp_client.SimpleUDPClient("127.0.0.1",spaceutil.resolumeOscPort)

# Activate the clips in Resolume.
# IMPORTANT!! The clip MUST be layer1, so that the
# Osc enabling/disabling of FFGL plugins works as intended.

print("Sleeping for ",resolumedelay," before sending OSC to activate Resolume")
sleep(resolumedelay)

for i in range(1,resolumelayers+1):
    resolume.send_message("/composition/layers/%d/clips/1/connect"%i,[1])

print("Resolume should now be running.")

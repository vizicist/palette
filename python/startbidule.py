import sys
import os
import json
import signal
import socket

from time import sleep
from subprocess import call, Popen
from pythonosc import udp_client

import spaceutil

signal.signal(signal.SIGINT, signal.SIG_IGN)

palettedir = os.getenv("PALETTE")
palettedirbackslash = palettedir.replace("/","\\")

paletteconfig = os.getenv("PALETTECONFIG")
if paletteconfig == "":
    paletteconfig = os.path.join(palettedirbackslash,"config")

plogue = udp_client.SimpleUDPClient("127.0.0.1",3210)
bidulepatch = os.path.join(paletteconfig,"palette.bidule")

print("Starting Bidule on ",bidulepatch)

dummy = Popen([
    "C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_x64.exe", bidulepatch])

# Wait for Bidule to completely load
print("Sleeping before turning Bidule off/on")
sleep(50)
print("Sending OSC to turn Bidule off/on")

# Turn audio off and back on (audio stutters if I don't do this?)
# print("Turning audio off")
plogue.send_message("/play",[0])
sleep(1.0)
plogue.send_message("/play",[1])
sleep(2.0)
plogue.send_message("/play",[1])
sleep(5.0)
plogue.send_message("/play",[1])
sleep(10.0)
plogue.send_message("/play",[1])


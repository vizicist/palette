import os
import shutil
import urllib
import urllib2
import json
from subprocess import call, Popen
from time import sleep
from nosuch.oscutil import *

def killtask(nm):
	call(["c:/windows/system32/taskkill","/f","/im",nm])

def mmtt_action(meth):
	url = 'http://127.0.0.1:4444/dojo.txt'
	params = '{}'
	id = '12345'
	data = '{ "jsonrpc": "2.0", "method": "'+meth+'", "params": "'+params+'", "id":"'+id+'" }\n'
	req = urllib2.Request(url, data)
	response = urllib2.urlopen(req)
	r = response.read()
	j = json.loads(r)
	if "result" in j:
		return j["result"]
	else:
		print "No result in JSON response!?  r="+r
		return -1

call(["c:/local/bin/tvon.bat"])
call(["c:/python27/python.exe","c:/local/manifold/bin/killall.py"])
call(["c:/python27/python.exe","c:/local/manifold/bin/debugcycle.py"])

mmtt_exe = "mmtt_kinetic.exe"
mmtt_exe = "mmtt_pcx.exe"
mmtt_exe = "mmtt_depth.exe"

mmtt_depth = Popen([mmtt_exe])
sleep(2)  # let it get running
try:
	while True:
		if mmtt_action("align_isdone") == 1:
			break
		sleep(1)
except:
	print "Hey, error from mmtt_action, ignoring??"

print "MMTT has finished aligning."

loopmidi = Popen(["/Program Files (x86)/Tobias Erichsen/loopMIDI/loopMIDI.exe"])

print "loopMIDI has been started."
sleep(1)

Popen(["c:/python27/python.exe","c:/local/manifold/bin/splash.py","","Loading..."])


bidule = Popen([
 	"C:\\Program Files\\Plogue\\Bidule\\PlogueBidule_x64.exe",
 	"\\local\\manifold\\patches\\bidule\\Palette_Alchemy_Burn.bidule"])
 
# Wait for Bidule to completely load all the Alchemy instances
sleep(50)

### patches="c:\\local\\manifold\\bin\\config\\palette"
### shutil.copy(patches+"\\default_burn.mnf",patches+"\\default.mnf")

arena = Popen(["C:\\Program Files (x86)\\Resolume Avenue 4.1.11\\Avenue.exe"])
 
## cd \local\python\nosuch_oscutil

global resolume
resolume = OscRecipient("127.0.0.1",7000)

# Activate the clips in Resolume.
# IMPORTANT!! The last clip activated MUST be layer1, so that the
# Osc enabling/disabling eof FFGL plugins works as intended.

sleep(12)

print "Sending OSC to activate Resolume."
resolume.sendosc("/layer2/clip1/connect",[1])
resolume.sendosc("/layer1/clip1/connect",[1])

# call(["c:/local/bin/nircmd.exe","win","settopmost","title","MMTT","1"])
# call(["c:/local/bin/nircmd.exe","win","max","title","MMTT"])

# Keep sending - Resolume might not be up yet
for i in range(5):
	sleep(2)
	resolume.sendosc("/layer2/clip1/connect",[1])
	resolume.sendosc("/layer1/clip1/connect",[1])

call(["c:/local/bin/nircmd.exe","win","setsize","title","MMTT","250","250","500","400"])
call(["c:/local/bin/nircmd.exe","win","settopmost","title","MMTT","1"])
# call(["c:/local/bin/nircmd.exe","win","max","title","MMTT","1"])

call(["c:/local/bin/nircmd.exe","win","min","stitle","Plogue"])


print "DONE!"

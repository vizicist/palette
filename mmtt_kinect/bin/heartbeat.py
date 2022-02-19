# Heartbeat utility for Space Palette
#
# Starts up all the Space Palette processes (MMTT, Plogue, Arena)
# and then monitors MIDI input (from the Logidy pedal) to do things
# like re-align, soft reset (restart processes), and hard reset (reboot windows)

import urllib
import urllib2
import json
from subprocess import call, Popen
from time import sleep
from traceback import format_exc
from threading import Thread, Lock

from nosuch.oscutil import *
from nosuch.midiutil import *
from nosuch.mididebug import *
from nosuch.midipypm import *

sayit = False

# Send a JSON message to MMTT
def mmtt_action(meth):
	try:
		print "DOACTION meth=",meth
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
	except:
		print "Unexpected exception in mmtt_action: %s" % format_exc()

def restart():
	print "restart() called"
	say("The Space Palette's initialization is underway")

	# This invokes startall.py synchronously, waiting for it to finish.
	# startall.py kills currently-running processes and restarts everything.
	call(["c:/python27/python.exe","c:/local/manifold/bin/startall.py"])

	say("The Space Palette initialization is complete")

def killall():
	call(["c:/python27/python.exe","c:/local/manifold/bin/killall.py"])

def reboot():
	say("Full reboot is underway")
	killall()
	# call(["c:/windows/system32/shutdown.exe","-t","0","-r","-f"])
	call(["c:/local/manifold/bin/reboot.bat"])

def reset():
	# print "reset() called"

	say("Please step away, registration is underway")
	mmtt_action("align_start")

	# print "Doing audio off/on"
	audio_off()
	sleep(2)
	audio_on()

	# Don't wait around, mmtt might not be alive
	# while True:
	# 	if mmtt_action("align_isdone") == 1:
	# 		break
	# 	sleep(1)
	# say("registration complete")

global PedalDown, PedalName

# PedalDown, if non-0, is the time at which it went down
PedalDown = {"LEFT":0.0, "MIDDLE":0.0, "RIGHT":0.0}
PedalName = {60:"LEFT", 62:"MIDDLE", 64:"RIGHT"}
PedalWaiter = {"LEFT":None, "MIDDLE":None, "RIGHT":None}

global Plogue
plogue = OscRecipient("127.0.0.1",3210)

def audio_off():
	plogue.sendosc("/play",[0])

def audio_on():
	plogue.sendosc("/play",[1])

def say(text):
	# Speaking works, but the audio starts out really quiet.
	# Disable it for the moment.
	if sayit:
		audio_off()
		call(["c:/local/bin/nircmd.exe","speak","text",text])
		audio_on()

class WaitForUp(Thread):
	def __init__(self,pedal,timeout,func):
		Thread.__init__(self)
		self.pedal = pedal
		self.timeout = timeout
		self.func = func

	def run(self):
		# print "WaitForUp start pedal=",self.pedal
		while True:
			sleep(0.2)
			down = PedalDown[self.pedal]
			if down == 0.0:
				break
			if down > 0.0 and (time.time()-down) > self.timeout:
				self.func()
				break
		PedalWaiter[self.pedal] = None
		print "WaitForUp end pedal=",self.pedal

def mymidi(e,d):
	m = e.midimsg
	# print("m=",m)

	if isinstance(m,NoteOff):
		if m.pitch in PedalName:
			nm = PedalName[m.pitch]
			print nm+" PEDAL UP"
			PedalDown[nm] = 0.0

	elif isinstance(m,NoteOn):
		if not m.pitch in PedalName:
			return

		nm = PedalName[m.pitch]
		PedalDown[nm] = time.time()
		print nm+" PEDAL DOWN = ",PedalDown[nm]

		if PedalWaiter[nm] != None:
			print nm+" PEDAL ALREADY DOWN!"
			return

		w = None
		if nm == "LEFT":
			say("Hold for 5 seconds to reboot completely")
			w = WaitForUp(nm,5.0,reboot)
		elif nm == "MIDDLE":
			say("Hold for 3 seconds to restart")
			w = WaitForUp(nm,3.0,restart)
		elif nm == "RIGHT":
			# say("Hold for on seconds to reset - alignment and audio")
			w = WaitForUp(nm,1.0,reset)

		if w:
			PedalWaiter[nm] = w
			w.start()

if __name__ == '__main__':

	Midi.startup()

	m = MidiPypmHardware()

	a = m.input_devices()
	i = None
	lookfor = "Logidy UMI3"
	for nm in a:
		if nm == lookfor:
			i = m.get_input(nm)
			i.open()
	if i == None:
		print "Unable to find MIDI input: "+lookfor
	else:
		print "Successfully opened MIDI input: "+lookfor

	Midi.callback(mymidi,"dummy")

	restart()

	try:
		while True:
			sleep(1.0)
	except KeyboardInterrupt:
		print "Got KeyboardInterrupt!"
	except:
		print "Unexpected exception: %s" % format_exc()

	Midi.shutdown()

	print "End of midimon.py"

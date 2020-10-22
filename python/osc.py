"""
This module provides event support for Osc things.
"""

import time
import socket
import traceback
import re
import sys
import pythonosc

from traceback import format_exc
from threading import Thread
from time import sleep
from pythonosc import dispatcher
from pythonosc import osc_server
from pythonosc import udp_client
from typing import List, Any

global Showalive,Showfseq
Showalive = False
Showfseq = False
Showsource = False

def containsAny(str, set):
	return 1 in [c in str for c in set]

def listencallback(ev,d):
	if type(ev.oscmsg[0]) == type([]):
		# It's a bundle, handle each message separately
		for m in ev.oscmsg:
			handleonemessage(m)
	else:
		# It's a single message
		handleonemessage(ev.oscmsg)

def handleonemessage(m):
	if Showalive==False and len(m) >= 3 and m[2] == "alive":
		return
	if Showfseq==False and len(m) >= 3 and m[2] == "fseq":
		return
	if Showsource==False and len(m) >= 3 and m[2] == "source":
		return
	# global time0
	# print(("%8.3f " % (time.time()-time0)) + m.__str__())
	print(m)

def mycallback(address: str, *args: List[Any]) -> None:
        print("received ",address," args=",args)

def dolisten(porthost):
    (port,host) = unpackporthost(porthost)
    print("Listening on host=",host," port=",port)

    d = dispatcher.Dispatcher()
    d.map("/*",mycallback)

    server = osc_server.ThreadingOSCUDPServer((host,int(port)), d)
    server.serve_forever()

def unpackporthost(porthost):
	if porthost.find("@") < 0:
		port = porthost
		host = "127.0.0.1"
	else:
		port = re.compile(".*@").search(porthost).group()[:-1]
		host = re.compile("@.*").search(porthost).group()[1:]
	return (port,host)

def usage():
	print("")
	print("Usage:")
	print("        osc {send} {port@host} {msg_addr} {msg_args}")
	print("        osc {listen} {port@host}")
	sys.exit(1)

if __name__ == '__main__':

	if len(sys.argv) < 2:
		usage()

	command = sys.argv[1]

	global time0
	time0 = time.time();

	if command == "send":
		if len(sys.argv) < 3:
			usage()
		(port,host) = unpackporthost(sys.argv[2])
		oscaddr = sys.argv[3]
		
		n = 4
		oscmsg = []
		while n < len(sys.argv):
			s = sys.argv[n]
			if len(s) > 0 and s[0].isdigit():
				if containsAny(s,"."):
					oscmsg.append(float(s))
				else:
					oscmsg.append(int(s))
			else:
				oscmsg.append(s)
			n += 1
		
		r = udp_client.SimpleUDPClient(host,int(port))
		r.send_message(oscaddr,oscmsg)

	elif command == "listen":
		if len(sys.argv) < 3:
			usage()
		Showalive = False
		Showfseq = False
		Showsource = False
		dolisten(sys.argv[2])

	elif command == "listenall":
		if len(sys.argv) < 3:
			usage()
		Showalive = True
		Showfseq = True
		Showsource = True
		dolisten(sys.argv[2])

	else:
		print("Unrecognized command: "+command)
		usage()


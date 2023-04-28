from nosuch.midiutil import *
from nosuch.midifile import *
from traceback import format_exc
from time import sleep
import sys

if __name__ == '__main__':

	if len(sys.argv) < 3:
		print "Usage: midiplayn {filename1} {outputdevice1} [...]"
		sys.exit(1)

	Midi.startup()

	# If you want to get debug output (e.g. to see when
	# things are written to MIDI output), use this:
	# Midi.debug = True

	midiout = []
	phrases = []
	n = 1
	while (n+1) < len(sys.argv):
		fname = sys.argv[n]
		outname = sys.argv[n+1]
		p = Phrase.fromMidiFile(fname)
		phrases.append(p)
		print "fname=",fname," number of notes=",len(p)
		# Open MIDI output named
		try:
			o = MidiOutput(outname)
			o.open()
			midiout.append(o)
		except:
			print "Error opening output: ",outname
			Midi.shutdown()
			sys.exit(1)

		n += 2

	# We want to schedule everything relative to the same starting time
	starttime = Midi.time_now() + 1.0

	speedup = 1
	for n in range(0,len(phrases)):
		for nt in phrases[n]:
			nt.clocks /= speedup
			midiout[n].schedule(nt,time=starttime)

	# Schedule the notes of the midifile on the output

	# sleep/wait till all the notes are played
	try:
		while len(Midi.scheduled()) > 0:
			sleep(1.0)
	except KeyboardInterrupt:
		print "Got KeyboardInterrupt!"
	except:
		print "Unexpected exception: %s" % format_exc()

	Midi.shutdown()

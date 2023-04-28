from nosuch.midiutil import *
from nosuch.midifile import *
from nosuch.midipypm import *
from traceback import format_exc
from time import sleep
import sys

if __name__ == '__main__':

	if len(sys.argv) < 1:
		print "Usage: midinote [ {outputdevice} ] "
		sys.exit(1)
	
	outname = sys.argv[1]

	Midi.startup()

	m = MidiPypmHardware()

	# Open MIDI output named
	try:
		# if outname is None, it opens the default output
		o = m.get_output(outname)
		o.open()
	except:
		print "Error opening output: %s, exception: %s" % (outname,format_exc())
		Midi.shutdown()
		sys.exit(1)

	p = Phrase()
	nt = NoteOn(pitch=64, channel=1, velocity=100)
	p.append(SequencedMidiMsg(nt,clocks=0))
	nt = NoteOff(pitch=64, channel=1, velocity=0)
	p.append(SequencedMidiMsg(nt,clocks=100))

	# Schedule the notes of the midifile on the output
	for n in p:
		o.schedule(n)

	# sleep/wait till all the notes are played
	try:
		while Midi.num_scheduled() > 0:
			sleep(1.0)
	except KeyboardInterrupt:
		print "Got KeyboardInterrupt!"
	except:
		print "Unexpected exception: %s" % format_exc()

	Midi.shutdown()

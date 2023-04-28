from nosuch.midiutil import *
from nosuch.midifile import *
from nosuch.oscutil import *
from nosuch.midiosc import *
from traceback import format_exc
from time import sleep
import sys

if __name__ == '__main__':

	if len(sys.argv) < 2 or len(sys.argv) > 4:
		print "Usage: oscplay {filename} {osc-output} {osc-input}"
		sys.exit(1)

	fname = sys.argv[1]
	output_name = sys.argv[2]
	if len(sys.argv) < 4:
		input_name = None
	else:
		input_name = sys.argv[3]

	Midi.startup()

	m = MidiOscHardware(input_name,output_name)

	# If you want to get debug output (e.g. to see when
	# things are written to MIDI output), use this:
	# Midi.debug = True

	# Read the midifile
	p = Phrase.fromMidiFile(fname)

	# Open MIDI output
	try:
		o = m.get_output()
		o.open()
	except:
		print "Error opening OSC output, exception: %s" % (format_exc())
		Midi.shutdown()
		sys.exit(1)

	# Schedule the notes of the midifile on the output
	for n in p:
		o.schedule(n)

	# sleep/wait till all the notes are played
	try:
		while len(Midi.scheduled()) > 0:
			sleep(1.0)
	except KeyboardInterrupt:
		print "Got KeyboardInterrupt!"
	except:
		print "Unexpected exception: %s" % format_exc()

	Midi.shutdown()

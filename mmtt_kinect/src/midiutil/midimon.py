from nosuch.midiutil import *
from traceback import format_exc
from time import sleep
from nosuch.mididebug import *
from nosuch.midipypm import *

if __name__ == '__main__':

	Midi.startup()

	# m = MidiDebugHardware()
	m = MidiPypmHardware()

	a = m.input_devices()
	for nm in a:
		print "Opening input = ",nm
		i = m.get_input(nm)
		i.open()

	def print_midi(msg,data):
		print("MIDI = %s" % str(msg))

	Midi.callback(print_midi,"dummy")

	try:
		while True:
			sleep(1.0)
	except KeyboardInterrupt:
		print "Got KeyboardInterrupt!"
	except:
		print "Unexpected exception: %s" % format_exc()

	Midi.shutdown()

	print "End of midimon.py"

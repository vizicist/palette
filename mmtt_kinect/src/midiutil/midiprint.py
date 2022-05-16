from nosuch.midiutil import *
from nosuch.midifile import *

if __name__ == '__main__':

	if len(sys.argv) < 2:
		print "Usage: midiprint {filename}"
		sys.exit(1)

	fname = sys.argv[1]

	# Read the midifile
	p = Phrase.fromMidiFile(fname)

	# Schedule the notes of the midifile on the output
	for n in p:
		print n

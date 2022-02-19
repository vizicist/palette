#include "NosuchUtil.h"
#include "NosuchException.h"
#include "NosuchMidi.h"
#include <string>

static char *ReadableMidiPitches[128];
static char *ReadableCanonic[12] = {
	"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"
};
static bool ReadableNotesInitialized = false;

bool NosuchDebugMidiAll = false;
bool NosuchDebugMidiNotes = false;

int QuarterNoteClicks = 96;

char* ReadableMidiPitch(int p) {
	if ( ! ReadableNotesInitialized ) {
		for ( int n=0; n<128; n++ ) {
			int canonic = n % 12;
			int octave = (n / 12) - 2;
			std::string s = NosuchSnprintf("%s%d",ReadableCanonic[canonic],octave);
			ReadableMidiPitches[n] = _strdup(s.c_str());
		}
		ReadableNotesInitialized = true;
	}
	return ReadableMidiPitches[p];
}

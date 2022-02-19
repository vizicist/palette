#ifndef NOSUCHMIDI_H
#define NOSUCHMIDI_H

#include "portmidi.h"
#include "NosuchException.h"

#define MIDI_CLOCK      0xf8
#define MIDI_ACTIVE     0xfe
#define MIDI_STATUS_MASK 0x80
#define MIDI_SYSEX      0xf0
#define MIDI_EOX        0xf7
#define MIDI_START      0xFA
#define MIDI_STOP       0xFC
#define MIDI_CONTINUE   0xFB
#define MIDI_F9         0xF9
#define MIDI_FD         0xFD
#define MIDI_RESET      0xFF
#define MIDI_NOTE_ON    0x90
#define MIDI_NOTE_OFF   0x80
#define MIDI_CHANNEL_AT 0xD0
#define MIDI_POLY_AT    0xA0
#define MIDI_PROGRAM    0xC0
#define MIDI_CONTROL    0xB0
#define MIDI_PITCHBEND  0xE0
#define MIDI_MTC        0xF1
#define MIDI_SONGPOS    0xF2
#define MIDI_SONGSEL    0xF3
#define MIDI_TUNE       0xF6

extern int QuarterNoteClicks;
extern bool NosuchDebugMidiAll;
extern bool NosuchDebugMidiNotes;
// #define QuarterNoteClicks 96

typedef long MidiTimestamp;

#define NO_VALUE -1

char* ReadableMidiPitch(int pitch);

class MidiMsg {
public:
	MidiMsg() {
		NosuchDebug(2,"MidiMsg constructor!");
		next = NULL;
	}
	virtual ~MidiMsg() {
		NosuchDebug(2,"MidiMsg destructor! this=%d",this);
		if ( next != NULL ) {
			NosuchDebug(2,"  MidiMsg destructor is deleting next=%d",next);
			delete next;
			next = NULL;
		}
	}
	virtual std::string DebugString() = 0;
	virtual PmMessage PortMidiMessage() = 0;
	virtual PmMessage PortMidiMessageOff() { return 0; }
	virtual int MidiType() { return -1; }
	virtual int Channel() { return -1; }
	virtual int Pitch() { return -1; }
	virtual void Transpose(int dp) { }
	virtual int Velocity() { return -1; }
	virtual int Controller() { return -1; }
	virtual int Value(int val = NO_VALUE) { return NO_VALUE; }
	bool isSameAs(MidiMsg* m) {
		switch (MidiType()) {
		case MIDI_NOTE_ON:
		case MIDI_NOTE_OFF:
			if ( MidiType() == m->MidiType()
				&& Channel() == m->Channel()
				&& Pitch() == m->Pitch()
				&& Velocity() == m->Velocity() )
				return true;
			break;
		default:
			NosuchDebug("MidiMsg::isSameAs not implemented for MidiType=%d",MidiType());
			break;
		}
		return false;
	}
	virtual MidiMsg* clone() {
		NosuchDebug("Unable to clone MidiMsg of type %d, returning NULL",MidiType());
		return NULL;
	}

	MidiMsg* next;
};

class ChanMsg : public MidiMsg {
public:
	ChanMsg(int ch) : MidiMsg() {
		NosuchAssert(ch>=1 && ch<=16);
		NosuchDebug(2,"ChanMsg constructor");
		_chan = ch;
	}
	virtual ~ChanMsg() {
		NosuchDebug(2,"ChanMsg destructor");
	}
	virtual std::string DebugString() = 0;
	virtual PmMessage PortMidiMessage() = 0;
	virtual PmMessage PortMidiMessageOff() { return 0; }
	virtual int MidiType() { return -1; }
	virtual int Pitch() { return -1; }
	virtual int Velocity() { return -1; }
	virtual int Controller() { return -1; }
	virtual int Value(int v = NO_VALUE) { return NO_VALUE; }
	int Channel() { return _chan; }
protected:
	int _chan;   // 1-based
};

class MidiNoteOff : public ChanMsg {
public:
	static MidiNoteOff* make(int ch, int p, int v) {
		MidiNoteOff* m = new MidiNoteOff(ch,p,v);
		NosuchDebug(2,"MidiNoteOff::make m=%d",m);
		return m;
	};
	std::string DebugString() {
		std::string s = "NoteOff "+NoteString();
		return s;
	}
	std::string NoteString() {
		std::string s = NosuchSnprintf("(ch=%d p=%s,p%d v=%d)",_chan,ReadableMidiPitch(_pitch),_pitch,_velocity);
		if ( next != NULL ) {
			MidiNoteOff* nextm = (MidiNoteOff*)next;
			NosuchAssert(nextm);
			s += nextm->NoteString();
		}
		return s;
	}
	PmMessage PortMidiMessage() {
		return Pm_Message(0x80 | (_chan-1), _pitch, _velocity);
	}
	int MidiType() { return MIDI_NOTE_OFF; }
	int Pitch() { return _pitch; }
	int Velocity() { return _velocity; }
	MidiNoteOff* clone() {
		MidiNoteOff* newm = MidiNoteOff::make(Channel(),Pitch(),Velocity());
		if ( next != NULL ) {
			MidiNoteOff* newnextm = (MidiNoteOff*)(next->clone());
			NosuchAssert(newnextm);
			newm->next = newnextm;
		}
		// NosuchDebug(2,"MidiNoteOff::clone this=%d newm=%d %s",(int)this,(int)newm,newm->DebugString().c_str());
		return newm;
	};
	void Transpose(int dp) {
		MidiNoteOff* m = (MidiNoteOff*)this;
		while ( m != NULL ) {
			m->_pitch += dp;
			m = (MidiNoteOff*)(m->next);
		}
	}
private:
	MidiNoteOff(int ch, int p, int v) : ChanMsg(ch) {
		_pitch = p;
		_velocity = v;
	};
	int _pitch;
	int _velocity;
};

class MidiNoteOn : public ChanMsg {
public:
	static MidiNoteOn* make(int ch, int p, int v) {
		MidiNoteOn* m = new MidiNoteOn(ch,p,v);
		NosuchDebug(2,"MidiNoteOn::make m=%d",m);
		return m;
	}
	MidiNoteOff* makenoteoff() {
		MidiNoteOff* newm = MidiNoteOff::make(Channel(),Pitch(),Velocity());
		return newm;
	}
	MidiNoteOn* clone() {
		MidiNoteOn* newm = MidiNoteOn::make(Channel(),Pitch(),Velocity());
		if ( next != NULL ) {
			MidiNoteOn* newnextm = (MidiNoteOn*)(next->clone());
			NosuchAssert(newnextm);
			newm->next = newnextm;
		}
		// NosuchDebug(2,"MidiNoteOn::clone this=%d newm=%d %s",(int)this,(int)newm,newm->DebugString().c_str());
		return newm;
	};
	void Transpose(int dp) {
		MidiNoteOn* m = (MidiNoteOn*)this;
		while ( m != NULL ) {
			m->_pitch += dp;
			m = (MidiNoteOn*)(m->next);
		}
	}
	PmMessage PortMidiMessage() {
		return Pm_Message(0x90 | (_chan-1), _pitch, _velocity);
	}
	PmMessage PortMidiMessageOff() {
		return Pm_Message(0x80 | (_chan-1), _pitch, 0);
	}
	std::string DebugString() {
		std::string s = "NoteOn "+NoteString();
		return s;
	}
	std::string NoteString() {
		std::string s = NosuchSnprintf("(ch=%d p=%s,p%d v=%d)",_chan,ReadableMidiPitch(_pitch),_pitch,_velocity);
		if ( next != NULL ) {
			MidiNoteOn* nextm = (MidiNoteOn*)next;
			NosuchAssert(nextm);
			s += nextm->NoteString();
		}
		return s;
	}
	int MidiType() { return MIDI_NOTE_ON; }
	int Pitch() { return _pitch; }
	int Velocity() { return _velocity; }
	int Velocity(int v) { _velocity = v; return _velocity; }
#if 0
	~MidiNoteOn() {
		NosuchDebug(2,"MidiNoteOn destructor");
	}
#endif
private:
	MidiNoteOn(int ch, int p, int v) : ChanMsg(ch) {
		NosuchDebug(2,"MidiNoteOn constructor");
		_pitch = p;
		_velocity = v;
	};
	int _pitch;
	int _velocity;
};

class MidiController : public ChanMsg {
public:
	static MidiController* make(int ch, int ctrl, int v) {
		extern bool DoControllers;
		if ( DoControllers ) {
			MidiController* m = new MidiController(ch,ctrl,v);
			NosuchDebug(2,"MidiController::make m=%d",m);
			return m;
		} else {
			NosuchDebug("Hey, MidiController::make shouldn't be called when !DoControllers");
			return NULL;
		}
	};
	std::string DebugString() {
		return NosuchSnprintf("Controller ch=%d ct=%d v=%d",_chan,_controller,_value);
	}
	PmMessage PortMidiMessage() {
		return Pm_Message(0xb0 | (_chan-1), _controller, _value);
	}
	int MidiType() { return MIDI_CONTROL; }
	int Controller() { return _controller; }
	int Value(int v = NO_VALUE) {
		if ( v >= 0 ) {
			NosuchAssert(v <= 127);
			_value = v;
		}
		return _value;
	}
	MidiController* clone() {
		MidiController* newm = MidiController::make(Channel(),Controller(),Value());
		// NosuchDebug(2,"MidiController::clone this=%d newm=%d %s",(int)this,(int)newm,newm->DebugString().c_str());
		return newm;
	};
private:
	MidiController(int ch, int ctrl, int v) : ChanMsg(ch) {
		_controller = ctrl;
		_value = v;
	};
	int _controller;
	int _value;
};

class MidiProgramChange : public ChanMsg {
public:
	static MidiProgramChange* make(int ch, int v) {
		MidiProgramChange* m = new MidiProgramChange(ch,v);
		NosuchDebug(2,"MidiProgramChange::make m=%d",m);
		return m;
	};
	std::string DebugString() {
		return NosuchSnprintf("ProgramChange ch=%d v=%d",_chan,_value);
	}
	PmMessage PortMidiMessage() {
		// Both channel and value going out are 0-based
		return Pm_Message(0xc0 | (_chan-1), _value-1, 0);
	}
	int MidiType() { return MIDI_PROGRAM; }
	int Value(int v = NO_VALUE) {
		if ( v > 0 ) {
			NosuchAssert(v<=128);  // program change value is 1-based
			_value = v;
		}
		return _value;
	}
	MidiProgramChange* clone() {
		MidiProgramChange* newm = MidiProgramChange::make(Channel(),Value());
		// NosuchDebug(2,"MidiProgramChange::clone this=%d newm=%d %s",(int)this,(int)newm,newm->DebugString().c_str());
		return newm;
	};
private:
	MidiProgramChange(int ch, int v) : ChanMsg(ch) {
		NosuchAssert(v>0);  // program change value is 1-based
		_value = v;
	};
	int _value;   // 1-based
};

class MidiPitchBend : public ChanMsg {
public:
	static MidiPitchBend* make(int ch, int v) {
		// The v coming in is expected to be 0 to 16383, inclusive
		MidiPitchBend* m = new MidiPitchBend(ch,v);
		return m;
	};
	std::string DebugString() {
		return NosuchSnprintf("PitchBend ch=%d v=%d",_chan,_value);
	}
	PmMessage PortMidiMessage() {

// The two bytes of the pitch bend message form a 14 bit number, 0 to 16383.
// The value 8192 (sent, LSB first, as 0x00 0x40), is centered, or "no pitch bend."
// The value 0 (0x00 0x00) means, "bend as low as possible,"
// and, similarly, 16383 (0x7F 0x7F) is to "bend as high as possible."

		NosuchAssert(_value >= 0 && _value <= 16383);
		return Pm_Message(0xe0 | (_chan-1), _value & 0x7f, (_value>>7) & 0x7f);
	}
	int MidiType() { return MIDI_PITCHBEND; }
	int Value(int v = NO_VALUE) {
		if ( v >= 0 ) {
			NosuchAssert(v >= 0 && v <= 16383);
			_value = v;
		}
		return _value;
	}
	MidiPitchBend* clone() {
		MidiPitchBend* newm = MidiPitchBend::make(Channel(),Value());
		// NosuchDebug(2,"MidiPitchBend::clone this=%d newm=%d %s",(int)this,(int)newm,newm->DebugString().c_str());
		return newm;
	};
private:
	MidiPitchBend(int ch, int v) : ChanMsg(ch) {
		NosuchAssert(v >= 0 && v <= 16383);
		Value(v);
	};
	int _value;   // from 0 to 16383
};

#endif
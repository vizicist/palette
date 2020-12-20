package engine

import (
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
)

// NoteType is NOTE, NOTEON, NOTEOFF, or NOTEBYTES
type NoteType int

// These are the constant values of NoteType
const (
	NOTE         NoteType = iota // full note with duration
	NOTEON       NoteType = iota
	NOTEOFF      NoteType = iota
	CONTROLLER   NoteType = iota
	PROGCHANGE   NoteType = iota
	CHANPRESSURE NoteType = iota
	PITCHBEND    NoteType = iota
	NOTEBYTES    NoteType = iota
)

// Clicks is a time or duration value.
// NOTE: A Clicks value can be negative because
// it's sometimes relative to the starting time of a Phrase
type Clicks int64

// MaxClicks is the high-possible value for Clicks
const MaxClicks = Clicks(math.MaxInt64)

// XXX - probably could have a StepNum type to distinguish Clicks that are
// XXX - used as absolute time versus Clicks that are step numbers

// Phrase is a time-ordered list of Notes
// which are MIDI messages and other realtime events).
type Phrase struct {
	rwmutex   *sync.RWMutex
	firstnote *Note
	lastnote  *Note
	Length    Clicks
}

// Note should be an interface?

// Note is a single item in a Phrase
type Note struct {
	TypeOf   NoteType // NOTE, NOTEON, NOTEOFF, CONTROLLER, NOTEBYTES
	Clicks   Clicks   // nanoseconds
	Duration Clicks   // nanoseconds, when it's a NOTE
	Pitch    uint8    // 0-127
	Velocity uint8    // 0-127
	Sound    string
	bytes    []byte
	next     *Note
}

// Data1 xxx
func (n Note) Data1() uint8 {
	return n.Pitch
}

// Data2 xxx
func (n Note) Data2() uint8 {
	return n.Velocity
}

// NewPhrase returns a new Phrase
func NewPhrase() *Phrase {
	return &Phrase{
		rwmutex: new(sync.RWMutex),
	}
}

// Lock for writing
func (p *Phrase) Lock() {
	p.rwmutex.Lock()
}

// Unlock for writing
func (p *Phrase) Unlock() {
	p.rwmutex.Unlock()
}

// RLock for reading
func (p *Phrase) RLock() {
	p.rwmutex.RLock()
}

// RUnlock for reading
func (p *Phrase) RUnlock() {
	p.rwmutex.RUnlock()
}

// Format xxx
func (n *Note) Format(f fmt.State, c rune) {
	p := &Phrase{}
	p.InsertNoLock(n)
	s := p.ToString()
	f.Write([]byte(s))
}

// NewNote create a new Note of type NOTE, i.e. with duration
func NewNote(pitch uint8, velocity uint8, duration Clicks, sound string) *Note {
	return &Note{TypeOf: NOTE, Pitch: pitch, Velocity: velocity, Duration: duration, Sound: sound}
}

// NewNoteOn create a new NOTEON
func NewNoteOn(pitch uint8, velocity uint8, sound string) *Note {
	return &Note{TypeOf: NOTEON, Pitch: pitch, Velocity: velocity, Sound: sound}
}

// NewNoteOff create a new NOTEOFF
func NewNoteOff(pitch uint8, velocity uint8, sound string) *Note {
	return &Note{TypeOf: NOTEOFF, Pitch: pitch, Velocity: velocity, Sound: sound}
}

// NewController create a new NOTEOFF
func NewController(controller uint8, value uint8, sound string) *Note {
	return &Note{TypeOf: CONTROLLER, Pitch: controller, Velocity: value, Sound: sound}
}

// NewProgChange xxx
func NewProgChange(program uint8, value uint8, sound string) *Note {
	return &Note{TypeOf: PROGCHANGE, Pitch: program, Velocity: value, Sound: sound}
}

// NewChanPressure xxx
func NewChanPressure(data1 uint8, velocity uint8, sound string) *Note {
	return &Note{TypeOf: CHANPRESSURE, Pitch: data1, Velocity: velocity, Sound: sound}
}

// NewPitchBend xxx
func NewPitchBend(data1 uint8, data2 uint8, sound string) *Note {
	return &Note{TypeOf: PITCHBEND, Pitch: data1, Velocity: data2, Sound: sound}
}

// EndOf returns the ending time of a note
func (n *Note) EndOf() Clicks {
	if n.TypeOf == NOTE {
		return n.Clicks + n.Duration
	}
	return n.Clicks
}

// IsNote returns true if the note is a NOTE, NOTEON, or NOTEOFF
func (n *Note) IsNote() bool {
	if n.TypeOf == NOTE || n.TypeOf == NOTEON || n.TypeOf == NOTEOFF {
		return true
	}
	return false
}

// ReadablePitch returns a readable string for a note pitch
// Note that it also includes a + or - if it's a NOTEON or NOTEOFF.
// If it's not a NOTE-type note, "" is returned
func (n *Note) ReadablePitch() string {
	scachars := []string{
		"c", "c+", "d", "e-", "e", "f", "f+",
		"g", "a-", "a", "b-", "b", "c",
	}

	pre := ""
	if n.TypeOf == NOTEON {
		pre = "+"
	} else if n.TypeOf == NOTEOFF {
		pre = "-"
	} else if n.TypeOf != NOTE {
		return ""
	}
	return fmt.Sprintf("%s%s", pre, scachars[n.Pitch%12])
}

// Compare is used to determine Note ordering
func (n *Note) Compare(n2 *Note) int {

	// Compare attributes in the following order:
	// Clicks, Typeof, Pitch, Sound, Velocity, Duration
	if d := n.Clicks - n2.Clicks; d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if d := n.TypeOf - n2.TypeOf; d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if d := n.Pitch - n2.Pitch; d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if d := strings.Compare(n.Sound, n2.Sound); d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if d := n.Velocity - n2.Velocity; d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if n.TypeOf == NOTE {
		if d := n.Duration - n2.Duration; d < 0 {
			return -1
		} else if d > 0 {
			return 1
		}
	}

	return 0
}

// Copy a Note.  NOTE: the next value is cleared
func (n *Note) Copy() *Note {
	newn := &Note{
		TypeOf:   n.TypeOf,
		Clicks:   n.Clicks,
		Duration: n.Duration,
		Pitch:    n.Pitch,
		Velocity: n.Velocity,
		Sound:    n.Sound,
		bytes:    n.bytes,
		next:     nil,
	}
	return newn
}

/*
OLD VERSION OF TOSTRING
// ToString returns a human-readable version of a Phrase
func (p *Phrase) ToString() string {
	s := "'"
	var first = true

	var lastClicks Clicks
	var lastDuration Clicks
	var lastEndClicks Clicks
	_ = lastEndClicks // why do I need this?  lastEndClicks is used
	var lastSound string

	lastClicks = 0
	lastDuration = math.MaxInt64
	lastEndClicks = 0

	for n := p.firstnote; n != nil; {
		// Put out the time value
		if first {
			first = false
		} else {
			// Separator is a space if it starts at the same time as the last one, otherwise comma
			if n.Clicks == lastClicks {
				s += " "
			} else {
				s += ","
			}
		}

		switch n.TypeOf {
		case NOTE:
			s += fmt.Sprintf("p%dv%d", n.Pitch, n.Velocity)
			if n.Duration != lastDuration {
				s += fmt.Sprintf("d%d", n.Duration)
				lastDuration = n.Duration
			}
		case NOTEON:
			s += fmt.Sprintf("+p%dv%d", n.Pitch, n.Velocity)
		case NOTEOFF:
			s += fmt.Sprintf("-p%dv%d", n.Pitch, n.Velocity)
		case CONTROLLER:
			s += fmt.Sprintf("x%02x%02x", n.Data1(), n.Data2())
		case PROGCHANGE:
			s += fmt.Sprintf("x%02x%02x", n.Data1(), n.Data2())
		case CHANPRESSURE:
			s += fmt.Sprintf("x%02x%02x", n.Data1(), n.Data2())
		case PITCHBEND:
			s += fmt.Sprintf("x%02x%02x", n.Data1(), n.Data2())
		case NOTEBYTES:
			return "<NOTEBYTES not yet handled>"
		default:
			return fmt.Sprintf("<Note Type %d not yet handled>", n.TypeOf)
		}

		if n.Clicks != lastClicks {
			s += fmt.Sprintf("t%d", n.Clicks)
			lastClicks = n.Clicks
		}
		if n.Sound != lastSound {
			lastSound = n.Sound
			s += fmt.Sprintf("S%s", n.Sound)
		}

		lastEndClicks = n.EndOf()
		n = n.next
	}
	s += "'"
	return s
}
*/

// ToString produces a human-readable version of a Note.
// Note that it includes the surrounding quotes that make it look like a Phrase
func (n *Note) ToString() string {

	pitch := n.ReadablePitch()
	if pitch == "" {
		log.Printf("Note.ToString unable to handle n.Typeof=%d\n", n.TypeOf)
		return "''"
	}
	octave := -2 + int(n.Pitch)/12 // MIDI octave
	s := fmt.Sprintf("'%so%d", pitch, octave)
	if n.TypeOf == NOTE {
		s += fmt.Sprintf("d%d", n.Duration)
	}
	s += fmt.Sprintf("v%dt%dS%s'", n.Velocity, n.Clicks, n.Sound)
	return s
}

////////////////////// Phrase methods /////////////////////////

// NumNotes returns the number of notes in a Phrase
func (p *Phrase) NumNotes() int {

	p.RLock()
	defer p.RUnlock()

	nnotes := 0
	for n := p.firstnote; n != nil; n = n.next {
		nnotes++
	}
	return nnotes
}

// ToString returns a human-readable version of a Phrase
func (p *Phrase) ToString() string {

	p.RLock()
	defer p.RUnlock()

	s := "'"
	var first = true

	var lastSound string

	lastClicks := Clicks(0)
	lastVelocity := uint8(0)
	lastDuration := Clicks(0)
	lastOctave := 0

	for n := p.firstnote; n != nil; {

		includeTime := true
		if !first {
			// Separator is a space if it starts at the same time as the last one, otherwise comma
			if n.Clicks == lastClicks {
				s += " "
				includeTime = false
			} else {
				s += ","
				// If the clicks+duration of the previous note are equal to the clicks of this note,
				// then we can omit the explicit time.  I.e. a comma means the the default
				// time of the next note is the end of the previous note.
				if n.Clicks == (lastClicks + lastDuration) {
					includeTime = false
				}
			}
		} else {
			// if first note is at time 0, don't includeTime
			if n.Clicks == 0 {
				includeTime = false
			}
		}

		pitch := n.ReadablePitch()
		if pitch == "" {
			log.Printf("Phrase.ToString unable to handle n.Typeof=%d, using c\n", n.TypeOf)
			pitch = "c"
		}
		s += pitch

		// MIDI octave
		octave := -2 + int(n.Pitch)/12
		if first || octave != lastOctave {
			s += fmt.Sprintf("o%d", octave)
			lastOctave = octave
		}

		if n.TypeOf == NOTE {
			if first || n.Duration != lastDuration {
				s += fmt.Sprintf("d%d", n.Duration)
			}
			lastDuration = n.Duration
		} else {
			lastDuration = 0
		}

		if first || n.Velocity != lastVelocity {
			s += fmt.Sprintf("v%d", n.Velocity)
			lastVelocity = n.Velocity
		}

		if includeTime {
			s += fmt.Sprintf("t%d", n.Clicks)
		}
		lastClicks = n.Clicks

		if n.Sound != lastSound {
			lastSound = n.Sound
			s += fmt.Sprintf("S%s", n.Sound)
		}

		n = n.next

		if first {
			first = false
		}
	}
	if p.lastnote != nil {
		s += fmt.Sprintf(",l%d", p.Length)
	}

	s += "'"
	return s
}

// Format lets you conveniently print a Phrase with fmt functions
func (p *Phrase) Format(f fmt.State, c rune) {
	f.Write([]byte(p.ToString()))
}

// ResetLengthNoLock sets the length of a Phrase to the end of the lastnote
func (p *Phrase) ResetLengthNoLock() {

	if p.lastnote == nil {
		p.Length = 0
	} else {
		n := p.lastnote
		if n.TypeOf == NOTE {
			p.Length = n.Clicks + n.Duration
		} else {
			p.Length = n.Clicks
		}
	}
}

// Append appends a note to the end of a Phrase, assuming that the last
// note in the Phrase is before or at the same time as tne appended note.
func (p *Phrase) Append(n *Note) {
	if p.firstnote == nil {
		p.firstnote = n
		p.lastnote = n
	} else {
		if p.lastnote.Clicks > n.Clicks {
			log.Printf("Hey, Append detects an out-of-order usage\n")
		}
		p.lastnote.next = n
		p.lastnote = n
	}
}

// InsertNoLock adds a Note to a Phrase
func (p *Phrase) InsertNoLock(note *Note) *Phrase {

	// log.Printf("Phrase.Insert note=%+v\n", note)
	if note.next != nil {
		log.Printf("Unexpected note.next!=nil in Phrase.InsertNoLock")
		return p
	}

	// Empty phrase, just set it
	if p.firstnote == nil {
		p.firstnote = note
		p.lastnote = note
		return p
	}

	if p.lastnote == nil {
		log.Printf("Expected lastnote to be nil when firstnote is nil!?")
		return p
	}

	// If it's after or equal to the last note, just append it
	if note.Compare(p.lastnote) >= 0 {
		p.lastnote.next = note
		p.lastnote = note
		return p
	}

	var prevnt *Note
	nt := p.firstnote
	// insert it just before the first note in the phrase that it is less-than
	for nt != nil {
		if note.Compare(nt) < 0 {
			// insert it before nt
			if prevnt == nil {
				note.next = p.firstnote
				p.firstnote = note
			} else {
				note.next = nt
				prevnt.next = note
			}
			return p
		}
		prevnt = nt
		nt = nt.next
	}
	return p
}

// InsertNote inserts a note into a Phrase
// NOTE: it's assumed that the Phrase is already locked for writing.
func (p *Phrase) InsertNote(nt *Note) *Phrase {
	return p.InsertNoLock(nt)
}

package engine

import (
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
)

// Clicks is a time or duration value.
// NOTE: A Clicks value can be negative because
// it's sometimes relative to the starting time of a Phrase.
// XXX - possibly could have a type to distinguish Clicks that are
// XXX - used as absolute time versus Clicks that are step numbers
type Clicks int64

// MaxClicks is the high-possible value for Clicks
const MaxClicks = Clicks(math.MaxInt64)

// Phrase is a time-ordered list of Notes
// which are MIDI messages and other realtime events).
type Phrase struct {
	rwmutex   *sync.RWMutex
	firstnote *Note
	lastnote  *Note
	Length    Clicks
}
type Phrasep *Phrase

// Note should be an interface?

// Note is a single item in a Phrase
type Note struct {
	Source   string // might be based on (or equal to) NUID
	TypeOf   string // note, noteon, noteoff, controller, notebytes
	Clicks   Clicks // nanoseconds
	Duration Clicks // nanoseconds, when it's a note
	Pitch    uint8  // 0-127
	Velocity uint8  // 0-127
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

// NewNote create a new Note of type note, i.e. with duration
func NewNote(pitch uint8, velocity uint8, duration Clicks, sound string) *Note {
	return &Note{TypeOf: "note", Pitch: pitch, Velocity: velocity, Duration: duration, Sound: sound}
}

// NewNoteOn create a new noteon
func NewNoteOn(pitch uint8, velocity uint8, sound string) *Note {
	return &Note{TypeOf: "noteon", Pitch: pitch, Velocity: velocity, Sound: sound}
}

// NewNoteOff create a new noteoff
func NewNoteOff(pitch uint8, velocity uint8, sound string) *Note {
	return &Note{TypeOf: "noteoff", Pitch: pitch, Velocity: velocity, Sound: sound}
}

// NewController create a new noteoff
func NewController(controller uint8, value uint8, sound string) *Note {
	return &Note{TypeOf: "controller", Pitch: controller, Velocity: value, Sound: sound}
}

// NewProgChange xxx
func NewProgChange(program uint8, value uint8, sound string) *Note {
	return &Note{TypeOf: "progchange", Pitch: program, Velocity: value, Sound: sound}
}

// NewChanPressure xxx
func NewChanPressure(data1 uint8, velocity uint8, sound string) *Note {
	return &Note{TypeOf: "chanpressure", Pitch: data1, Velocity: velocity, Sound: sound}
}

// NewPitchBend xxx
func NewPitchBend(data1 uint8, data2 uint8, sound string) *Note {
	return &Note{TypeOf: "pitchbend", Pitch: data1, Velocity: data2, Sound: sound}
}

// EndOf returns the ending time of a note
func (n *Note) EndOf() Clicks {
	if n.TypeOf == "note" {
		return n.Clicks + n.Duration
	}
	return n.Clicks
}

// IsNote returns true if the note is a note, noteon, or noteoff
func (n *Note) IsNote() bool {
	if n.TypeOf == "note" || n.TypeOf == "noteon" || n.TypeOf == "noteoff" {
		return true
	}
	return false
}

// ReadablePitch returns a readable string for a note pitch
// Note that it also includes a + or - if it's a noteon or noteoff.
// If it's not a NOTE-type note, "" is returned
func (n *Note) ReadablePitch() string {
	scachars := []string{
		"c", "c+", "d", "e-", "e", "f", "f+",
		"g", "a-", "a", "b-", "b", "c",
	}

	pre := ""
	if n.TypeOf == "noteon" {
		pre = "+"
	} else if n.TypeOf == "noteoff" {
		pre = "-"
	} else if n.TypeOf != "note" {
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

	if d := strings.Compare(n.TypeOf, n2.TypeOf); d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if d := int(n.Pitch) - int(n2.Pitch); d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if d := strings.Compare(n.Sound, n2.Sound); d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if d := int(n.Velocity) - int(n2.Velocity); d < 0 {
		return -1
	} else if d > 0 {
		return 1
	}

	if n.TypeOf == "note" {
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

// ToString produces a human-readable version of a Note.
// Note that it includes the surrounding quotes that make it look like a Phrase
func (n *Note) ToString() string {

	pitch := n.ReadablePitch()
	if pitch == "" {
		log.Printf("Note.ToString unable to handle n.Typeof=%s\n", n.TypeOf)
		return "''"
	}
	octave := -2 + int(n.Pitch)/12 // MIDI octave
	s := fmt.Sprintf("'%so%d", pitch, octave)
	if n.TypeOf == "note" {
		s += fmt.Sprintf("d%d", n.Duration)
	}
	s += fmt.Sprintf("v%dt%dS%s'", n.Velocity, n.Clicks, n.Sound)
	return s
}

func NoteFromString(s string) (*Note, error) {
	if s == "" {
		return nil, fmt.Errorf("NoteFromString: bad format - %s", s)
	}
	i := 0
	ntype := "note"
	switch s[i] {
	case '+':
		ntype = "noteon"
		i++
	case '-':
		ntype = "noteoff"
		i++
	}
	var pitch int
	if s[i] == 'p' {
		// scan number
		log.Printf("NoteFromString: needs implementation here\n")
		pitch = 33
		i++
	} else {
		letters := map[string]int{"a": 0, "b": 1, "c": 2, "d": 3, "e": 4, "f": 5, "g": 6}
		p, ok := letters[string(s[i])]
		if !ok {
			return nil, fmt.Errorf("NoteFromString: bad note letter - %s", s)
		}
		pitch = p
	}
	vel := 64
	note := &Note{
		Source:   "",           // might be based on (or equal to) NUID
		TypeOf:   ntype,        // note, noteon, noteoff, controller, notebytes
		Clicks:   0,            // nanoseconds
		Duration: 0,            // nanoseconds, when it's a note
		Pitch:    uint8(pitch), // 0-127
		Velocity: uint8(vel),   // 0-127
		Sound:    "",
	}
	return note, nil
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
			log.Printf("Phrase.ToString unable to handle n.Typeof=%s, using c\n", n.TypeOf)
			pitch = "c"
		}
		s += pitch

		// MIDI octave
		octave := -2 + int(n.Pitch)/12
		if first || octave != lastOctave {
			s += fmt.Sprintf("o%d", octave)
			lastOctave = octave
		}

		if n.TypeOf == "note" {
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
		if n.TypeOf == "note" {
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
		log.Printf("Expected lastnote to be not-nil when firstnote is not nil!?")
		// try to fix it up
		p.lastnote = p.firstnote
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

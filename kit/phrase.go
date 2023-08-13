package kit

import (
	"bufio"
	"container/list"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Phrase is a time-ordered list of *PhraseElements
type Phrase struct {
	rwmutex     sync.RWMutex
	list        *list.List
	Source      string
	Destination string
	Length      Clicks
}

type PhraseElement struct {
	AtClick Clicks
	Value   any // things like *NoteFull, *NoteOn, *NoteOff, etc
}

func (pe *PhraseElement) Copy() *PhraseElement {
	newpe := &PhraseElement{
		AtClick: pe.AtClick,
		Value:   pe.Value,
	}
	return newpe
}

// NewPhrase returns a new Phrase
func NewPhrase(vals ...string) *Phrase {
	p := &Phrase{
		list: list.New(),
		// rwmutex: new(sync.RWMutex),
	}
	if len(vals) > 0 {
		LogWarn("NewPhrase of constant needs work")
	}
	return p
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
func (pi *PhraseElement) Format(f fmt.State, c rune) {
	var valstr string
	switch v := pi.Value.(type) {
	case *NoteOn:
		valstr = v.String()
	default:
		valstr = "UNKNOWNTYPE"
	}
	final := fmt.Sprintf("(PhraseElement AtClick=%d Value=%s)", pi.AtClick, valstr)
	f.Write([]byte(final))
}

/*
// NewNote create a new Note of type note, i.e. with duration
func NewNote(pitch uint8, velocity uint8, duration Clicks, sound string) *Note {
	return &Note{TypeOf: "note", Pitch: pitch, Velocity: velocity, Duration: duration, Synth: sound}
}

func NewBytes(bytes []byte) *Note {
	return &Note{TypeOf: "notebytes", bytes: bytes}
}
*/

/*
command	meaning	# param
0xF0	start of system exclusive message	variable
0xF1	MIDI Time Code Quarter Frame (Sys Common)
0xF2	Song Position Pointer (Sys Common)
0xF3	Song Select (Sys Common)
0xF4	???
0xF5	???
0xF6	Tune Request (Sys Common)
0xF7	end of system exclusive message	0
0xF8	Timing Clock (Sys Realtime)
0xFA	Start (Sys Realtime)
0xFB	Continue (Sys Realtime)
0xFC	Stop (Sys Realtime)
0xFD	???
0xFE	Active Sensing (Sys Realtime)
0xFF	System Reset (Sys Realtime)
*/

// NewNoteOn create a new noteon
func NewNoteOn(synth *Synth, pitch, velocity uint8) *NoteOn {
	return &NoteOn{Pitch: pitch, Velocity: velocity, Synth: synth}
}

// NewNoteOff create a new noteoff
func NewNoteOff(synth *Synth, pitch, velocity uint8) *NoteOff {
	return &NoteOff{Pitch: pitch, Velocity: velocity, Synth: synth}
}

// NewNoteOff create a new NoteOff from a NoteOn
func NewNoteOffFromNoteOn(nt *NoteOn) *NoteOff {
	return &NoteOff{Pitch: nt.Pitch, Velocity: nt.Velocity, Synth: nt.Synth}
}

/*
// ReadablePitch returns a readable string for a note pitch
// Note that it also includes a + or - if it's a noteon or noteoff.
// If it's not a NOTE-type note, "" is returned
func (n *PhraseAble) ReadablePitch() string {
	scachars := []string{
		"c", "c+", "d", "e-", "e", "f", "f+",
		"g", "a-", "a", "b-", "b", "c",
	}

	pre := ""
	switch n.(type) {
	case NoteOn:
	}
	if n.TypeOf == "noteon" {
		pre = "+"
	} else if n.TypeOf == "noteoff" {
		pre = "-"
	} else if n.TypeOf != "note" {
		return ""
	}
	return fmt.Sprintf("%s%s", pre, scachars[n.Pitch%12])
}
*/

/*
// Compare is used to determine Note ordering
func (n Note) Compare(n2 *Note) int {

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

	if d := strings.Compare(n.Synth, n2.Synth); d < 0 {
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
func (n Note) Copy() *Note {
	Warn("Note.Copy needs work")
	return nil
	/*
		newn := &Note{
			TypeOf:   n.TypeOf,
			Clicks:   n.Clicks,
			Duration: n.Duration,
			Pitch:    n.Pitch,
			Velocity: n.Velocity,
			Synth:    n.Synth,
			bytes:    n.bytes,
			next:     nil,
		}
		return newn
}
*/

/*
// String produces a human-readable version of a Note.
// Note that it includes the surrounding quotes that make it look like a Phrase
func (n Note) String() string {

	pitch := n.ReadablePitch()
	if pitch == "" {
		Warn("Note.ToString unable to handle", "typeof", n.TypeOf)
		return "''"
	}
	octave := -2 + int(n.Pitch)/12 // MIDI octave
	s := fmt.Sprintf("'%so%d", pitch, octave)
	if n.TypeOf == "note" {
		s += fmt.Sprintf("d%d", n.Duration)
	}
	s += fmt.Sprintf("v%dt%d", n.Velocity, n.Clicks)
	if n.Synth != "" {
		s += fmt.Sprintf("S%s", n.Synth)

	}
	s += "'"
	return s
}
*/

func SchedElementFromString(s string) (se *SchedElement, err error) {

	if s == "" {
		return nil, fmt.Errorf("NoteOnOffFromString: bad format - %s", s)
	}
	ntype := "note"
	reader := strings.NewReader(s)
	scanner := NewPhraseScanner(reader)
	state := 0
	npitch := -1
	nflat := false
	nsharp := false
	// ndur := -1
	noctave := 3 // Note: default octave is 3
	atclick := Clicks(0)
	// nchannel := 0
	endstate := 99
	nattribute := ""
	synthName := "default"
	for state != endstate {

		// At the beginning (state 0) we only want 1 character
		switch state {
		case 0:
			ch := scanner.ScanChar()
			switch ch {
			case "":
				state = endstate
			case "'":
				continue
			case "-":
				ntype = "noteoff"
				// stay in state 0
			case "+":
				ntype = "noteon"
				// stay in state 0
			case "p":
				state = 1
			case "c":
				npitch = 24
				state = 2
			case "d":
				npitch = 26
				state = 2
			case "e":
				npitch = 28
				state = 2
			case "f":
				npitch = 29
				state = 2
			case "g":
				npitch = 31
				state = 2
			case "a":
				npitch = 33
				state = 2
			case "b":
				npitch = 35
				state = 2
			default:
				return nil, fmt.Errorf("unexpected in Phrase: ch=%s", ch)
			}

		case 1: // after 'p'
			npitch, err = scanner.ScanNumber()
			if err != nil {
				return nil, err
			}
			ch := scanner.ScanChar()
			switch ch {
			case "":
				state = endstate
			default:
				nattribute = ch
				state = 3
			}

		case 2: // after a note (a,b,c,d,...)
			ch := scanner.ScanChar()
			switch ch {
			case "", "\x00":
				state = endstate
			case "-":
				nflat = true
				// stay in state 2
			case "+":
				nsharp = true
				// stay in state 2
			case "d", "o", "v", "t", "S": // octave
				nattribute = ch
				state = 3
			default:
				return nil, fmt.Errorf("unexpected char: ch=%s", ch)
			}

		case 3:
			// we've seen a note attribute,
			// now scan whatever comes after it.
			switch nattribute {
			/*
				case "d":
					ndur, err = scanner.ScanNumber()
					if err != nil {
						return nil, err
					}
			*/

			case "o":
				noctave, err = scanner.ScanNumber()
				if err != nil {
					return nil, err
				}

				/*
					case "c": // channel
						nchannel, err = scanner.ScanNumber()
						if err != nil {
							return nil, err
						}
				*/

			case "S": // channel
				_, synthName = scanner.ScanWord()
				if err != nil {
					return nil, err
				}

			case "v": // velocity
				noctave, err = scanner.ScanNumber()
				if err != nil {
					return nil, err
				}

			case "t": // time
				number, err := scanner.ScanNumber()
				if err != nil {
					return nil, err
				}
				atclick = Clicks(number)

			default:
				return nil, fmt.Errorf("bad attribute: %s", nattribute)
			}
			// read the next char, either another attribute, or end
			nattribute = scanner.ScanChar()
			if nattribute == "" || nattribute == "'" {
				state = endstate
			}

		default:
			return nil, fmt.Errorf("bad state: %d", state)
		}
	}

	if nflat {
		npitch--
	}
	if nsharp {
		npitch++
	}
	npitch = npitch + noctave*12
	if npitch > 127 {
		return nil, fmt.Errorf("pitch > 127")
	}
	if npitch < 0 {
		return nil, fmt.Errorf("pitch < 0")
	}
	nvelocity := 64

	synth := GetSynth(synthName)

	var val any
	switch ntype {
	/*
		case "note":
			val = &NoteFull{
				Channel:  uint8(nchannel),
				Duration: Clicks(ndur),
				Pitch:    uint8(npitch),
				Velocity: uint8(nvelocity),
			}
	*/
	case "noteon":
		val = &NoteOn{
			Synth:    synth,
			Pitch:    uint8(npitch),
			Velocity: uint8(nvelocity),
		}
	case "noteoff":
		val = &NoteOff{
			Synth:    synth,
			Pitch:    uint8(npitch),
			Velocity: uint8(nvelocity),
		}
	default:
		err := fmt.Errorf("unknown ntype = %s", ntype)
		return nil, err
	}

	se = NewSchedElement(atclick,"fromString", val)
	return se, nil
}

/*
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

// String returns a human-readable version of a Phrase
func (p *Phrase) String() string {

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
			Warn("Phrase.ToString unable to handle, using c", "typeof", n.TypeOf)
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

		if n.Synth != lastSound {
			lastSound = n.Synth
			if n.Synth != "" {
				s += fmt.Sprintf("S%s", n.Synth)
			}
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
	f.Write([]byte(p.String()))
}
*/

// ResetLengthNoLock sets the length of a Phrase to the end of the lastnote
func (p *Phrase) ResetLengthNoLock() {

	lasti := p.list.Back()
	if lasti == nil {
		p.Length = 0
		return
	}
	pi := lasti.Value.(*PhraseElement)
	switch v := pi.Value.(type) {
	case NoteFull:
		p.Length = pi.AtClick + v.Duration
	case NoteOn, NoteOff:
		p.Length = pi.AtClick
	}
}

/*
// Append appends a note to the end of a Phrase, assuming that the last
// note in the Phrase is before or at the same time as tne appended note.
func (p *Phrase) Append(n *Note) {
	if p.firstnote == nil {
		p.firstnote = n
		p.lastnote = n
	} else {
		if p.lastnote.Clicks > n.Clicks {
			Warn("Hey, Append detects an out-of-order usage")
		}
		p.lastnote.next = n
		p.lastnote = n
	}
}
*/

func (pe *PhraseElement) EndOf() Clicks {
	endof := pe.AtClick
	switch v := pe.Value.(type) {
	case *NoteFull:
		endof += v.Duration
	}
	return endof
}

func (pe *PhraseElement) IsNote() bool {
	switch pe.Value.(type) {
	case *NoteOn, *NoteOff, *NoteFull:
		return true
	default:
		return false
	}
}

// InsertNoLock adds a Note to a Phrase
func (p *Phrase) InsertNoLock(pe *PhraseElement) *Phrase {

	if p.list.Front() == nil {
		p.list.PushFront(pe)
		return p
	}
	click := pe.AtClick

	// If it's after or equal to the last note, just append it
	laste := p.list.Back()
	lastpe := laste.Value.(*PhraseElement)
	lastclick := lastpe.AtClick

	if click >= lastclick {
		p.list.PushBack(pe)
		return p
	}

	for e := p.list.Front(); e != nil; e = e.Next() {
		thisClick := e.Value.(*PhraseElement).AtClick
		if click < thisClick {
			p.list.InsertBefore(pe, e)
			break
		}
	}
	return p
}

// Token is something returned by the PhraseScanner
type Token int

const (
	EOF Token = iota
	WORD
	NUMBER
	MINUS
	PLUS
	COMMA
	SINGLEQUOTE
	SPACE
	UNKNOWN
)

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isWord(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		(ch == '_')

}

func isNumber(ch rune) bool {
	return ch == '-' || (ch >= '0' && ch <= '9')
}

// InsertNote inserts a note into a Phrase
func (p *Phrase) InsertElement(item *PhraseElement) *Phrase {
	// XXX - should lock here?
	// p.Lock()
	// defer p.Unlock()
	return p.InsertNoLock(item)
}

// PhraseScanner represents a lexical scanner for phrase constants
type PhraseScanner struct {
	r *bufio.Reader
}

// NewScanner returns a new instance of Scanner.
func NewPhraseScanner(r io.Reader) *PhraseScanner {
	return &PhraseScanner{r: bufio.NewReader(r)}
}

// read reads the next rune from the bufferred reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (s *PhraseScanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return rune(0)
	}
	return ch
}

// unread places the previously read rune back on the reader.
func (s *PhraseScanner) unread() {
	_ = s.r.UnreadRune()
}

func (s *PhraseScanner) ScanChar() string {
	ch := s.read()
	return string(ch)
}

func (s *PhraseScanner) ScanNumber() (int, error) {
	tok, str := s.ScanWord()
	if tok != NUMBER {
		return 0, fmt.Errorf("unexpected non-NUMBER: str=%s", str)
	}
	n, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("bad NUMBER: str=%s", str)
	}
	return n, nil
}

// Scan returns the next token and literal value.
func (s *PhraseScanner) ScanWord() (tok Token, lit string) {
	// Read the next rune.
	ch := s.read()

	// If we see whitespace then consume all contiguous whitespace.
	// If we see a letter then consume as an ident or reserved word.
	if isWhitespace(ch) {
		for {
			ch = s.read()
			if !isWhitespace(ch) {
				s.unread()
				break
			}
		}
		return SPACE, ""
	} else if isNumber(ch) {
		word := string(ch)
		for {
			ch = s.read()
			if !isNumber(ch) {
				s.unread()
				break
			}
			word += string(ch)
		}
		return NUMBER, word
	} else if isWord(ch) {
		word := string(ch)
		for {
			ch = s.read()
			if !isWord(ch) {
				s.unread()
				break
			}
			word += string(ch)
		}
		return WORD, word
	}

	// Otherwise read the individual character.
	switch ch {
	case '\'':
		return SINGLEQUOTE, "'"
	case '-':
		return MINUS, "-"
	case '+':
		return PLUS, "+"
	case ',':
		return COMMA, ","
	default:
		return UNKNOWN, string(ch)
	}
}

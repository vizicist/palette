package engine

/*
import (
	"container/list"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// UnfinishedDuration is an 'unset' value for Duration
const UnfinishedDuration = math.MaxInt32 - 1

const defaultReleaseVelocity = byte(0)

// MIDIFile lets you read a MIDI File
type MIDIFile struct {
	path               string
	dat                []byte
	sofar              int
	toberead           int
	format             int
	ntracks            int
	division           int
	parsed             bool
	tracks             []*Phrase
	currentTrackPhrase *Phrase // used while reading the file
	phrase             *Phrase // all tracks merged into a single phrase
	clickfactor        float32
	currtime           int
	dosysexcontinue    bool
	bytes              []byte
	// noteq              *Phrase // noteons to be completed when noteoffs found
	noteq      *list.List
	onoffmerge bool
	numq       int
}

// NewMIDIFile creates a MIDIFile
func NewMIDIFile(path string) (*MIDIFile, error) {
	m := &MIDIFile{
		path:            path,
		dosysexcontinue: true,
		tracks:          make([]*Phrase, 0),
		onoffmerge:      true,
		numq:            0,
	}
	err := m.Parse()
	if err != nil {
		return nil, fmt.Errorf("error in MIDIFile.Phrase: %s", err)
	}
	return m, nil
}

// Phrase returns a single Phrase containing all tracks in the MIDIFile
func (m *MIDIFile) Phrase() *Phrase {
	if m.phrase == nil {
		m.phrase = NewPhrase()
		for n := 0; n < m.ntracks; n++ {
			m.phrase = m.phrase.Merge(m.tracks[n])
		}
	}
	return m.phrase
}

// Parse reads the contents of a MIDIFile and creates Phrases for each track
func (m *MIDIFile) Parse() error {
	if m.parsed {
		return nil
	}
	dat, err := os.ReadFile(m.path)
	if err != nil {
		return fmt.Errorf("unable to read midifile: path=%s err=%s", m.path, err)
	}
	m.dat = dat
	m.sofar = 0
	err = m.readHeader()
	if err != nil {
		return err
	}
	for len(m.tracks) < m.ntracks {
		err := m.readTrack()
		if err == io.EOF {
			// Normal EOF
			break
		} else if err != nil {
			return fmt.Errorf("error while reading track: %s", err)
		}
	}
	if len(m.tracks) != m.ntracks {
		return fmt.Errorf("ntracks in header (%d) doesn't match number of tracks read (%d)",
			m.ntracks, len(m.tracks))
	}

	// Expect an EOF if we try to read more
	if m.readTrack() != io.EOF {
		Warn("Hmmm, there's extra stuff in the MIDIFile after reading all tracks!")
	}

	m.parsed = true
	return nil
}

func (m *MIDIFile) readMT() (string, error) {
	b0, _ := m.readc()
	b1, _ := m.readc()
	b2, _ := m.readc()
	b3, err := m.readc()
	if err != nil {
		return "", err
	}
	s := string(b0) + string(b1) + string(b2) + string(b3)
	return s, nil
}

func (m *MIDIFile) readHeader() error {
	s, err := m.readMT()
	if err != nil {
		return err
	}
	if s != "MThd" {
		return fmt.Errorf("bad header in midifile, expecting MThd, got %s", s)
	}

	m.toberead, _ = m.read32bit()
	m.format, _ = m.read16bit()
	m.ntracks, _ = m.read16bit()
	m.division, err = m.read16bit()
	if err != nil {
		return err
	}

	var clicks float32 = 96 // constant, not based on defaultClicksPerSecond or anything that changes
	var tempo float32 = 500000
	if (0x8000 & m.division) != 0 {
		// It's SMPTE, frame-per-second and ticks per frame
		framesPerSecond := (m.division >> 8) & 0x7f
		ticksPerFrame := m.division & 0xff
		m.clickfactor = (float32)(framesPerSecond*ticksPerFrame) / (clicks * (1000000.0 / tempo))
	} else {
		m.clickfactor = (float32)(m.division) / clicks
	}

	// flush any extra, in case header length is not 6
	for m.toberead > 0 {
		m.readc()
	}

	return nil
}

func (m *MIDIFile) starttrack() {
	m.currentTrackPhrase = NewPhrase()
	m.noteq = list.New()
}

// output the top Noteq and remove it from the list
func (m *MIDIFile) putnfree() {
	n := m.noteq.Remove(m.noteq.Front())

	m.numq--

	if n.Duration == UnfinishedDuration {
		n.Duration = m.clicks() - n.Clicks
	}

	m.currentTrackPhrase = m.currentTrackPhrase.InsertNoLock(n)
	m.currentTrackPhrase.ResetLengthNoLock()
}

func (m *MIDIFile) putallnotes() {
	for m.noteq.Front() != nil {
		m.putnfree()
	}
}

func (m *MIDIFile) endtrack() {
	m.putallnotes()
	if m.currentTrackPhrase == nil {
		Warn("unexpected nil value of m.currentTrackPhrase")
	} else {
		m.tracks = append(m.tracks, m.currentTrackPhrase)
		m.currentTrackPhrase = nil
	}
}
func (m *MIDIFile) msginit() {
	m.bytes = []byte{}
}
func (m *MIDIFile) msgadd(c byte) {
	m.bytes = append(m.bytes, c)
}
func (m *MIDIFile) msgbytes() []byte {
	return m.bytes
}
func (m *MIDIFile) metaevent(metatype byte) {
}
func (m *MIDIFile) noteon(synth string, pitch, velocity byte) {
	if velocity == 0 {
		m.noteoff(synth, pitch, defaultReleaseVelocity)
	} else {
		m.queuenote(synth, pitch, velocity, "noteon")
	}
}

func (m *MIDIFile) noteoff(synth string, pitch, velocity byte) {

	// find the first note-on (if any) that matches this one
	n := m.noteq.Front()
	for ; n != nil; n = n.Next() {
		if n.Synth() == synth && n.Pitch() == pitch && n.TypeOf() == "noteon" {
			break
		}
	}
	if n == nil {
		// it's an isolated note-off
		n = m.queuenote(synth, pitch, velocity, "noteoff")
		n.Duration = 0
	} else if !m.onoffmerge && velocity != defaultReleaseVelocity {
		// If the note-off matches a previous note-on, but has a
		// non-default velocity, then we have to turn it into a
		// separate keykit note-off, instead of merging it with
		// the note-on into a single note.
		o := m.queuenote(synth, pitch, velocity, "noteoff")
		n.Duration = 0
		o.Duration = 0
	} else {
		// A completed note.
		n.TypeOf = "note"
		n.Duration = m.clicks() - n.Clicks

		// If the MIDI File contains negative delta times (which
		// probably aren't legal!) the duration turns out to be
		// negative.  Here we deal with that.
		if n.Duration < 0 {
			n.Clicks = n.Clicks + n.Duration
			n.Duration = -n.Duration
		}
	}

	// Now start at the beginning of the list and put out any
	// notes we've completed.  This guarantees that the starting
	// times of the notes are in the proper (ie. monotonically
	// progressing) order.

	for {
		n := m.noteq.Front()
		if n == nil {
			break
		}
		// quit when we get to the first unfinished note
		if n.TypeOf != "notebytes" && n.Duration == UnfinishedDuration {
			break
		}
		m.putnfree()
	}
	// If the number of notes int Noteq gets too big, then we're
	// probably suffering from a note-on that never had a note-off
	// Force it out.
	if m.numq > 1024 {
		m.putnfree()
	}
}

func (m *MIDIFile) pressure(synth string, c1, c2 byte) {
	m.queuebytes(synth, []byte{PressureStatus, c1, c2})
}

func (m *MIDIFile) controller(synth string, c1, c2 byte) {
	m.queuebytes(synth, []byte{ControllerStatus, c1, c2})
}

func (m *MIDIFile) pitchbend(synth string, c1, c2 byte) {
	m.queuebytes(synth, []byte{PitchbendStatus, c1, c2})
}

func (m *MIDIFile) program(synth string, c1 byte) {
	m.queuebytes(synth, []byte{ProgramStatus, c1})
}

func (m *MIDIFile) chanpressure(synth string, c1 byte) {
	m.queuebytes(synth, []byte{ChanPressureStatus, c1})
}

func (m *MIDIFile) queuebytes(synth string, bytes []byte) {
	n := &Note{
		TypeOf: "notebytes",
		Clicks: m.clicks(),
		bytes:  bytes,
		Synth:  synth,
		next:   nil,
	}
	m.add2noteq(n)
}

func (m *MIDIFile) sysex(synth string, bytes []byte) {
	m.queuebytes(synth, bytes)
}

func (m *MIDIFile) chanmessage(status, c1, c2 byte) {
	channel := status & 0xf
	synth := fmt.Sprintf("channel%d", channel+1)
	switch status & 0xf0 {
	case NoteOnStatus:
		m.noteon(synth, c1, c2)
	case NoteOffStatus:
		m.noteoff(synth, c1, c2)
	case PressureStatus:
		m.pressure(synth, c1, c2)
	case ControllerStatus:
		m.controller(synth, c1, c2)
	case PitchbendStatus:
		m.pitchbend(synth, c1, c2)
	case ProgramStatus:
		m.program(synth, c1)
	case ChanPressureStatus:
		m.chanpressure(synth, c1)
	}
}

func (m *MIDIFile) arbitrary(synth string, bytes []byte) {
	m.queuebytes(synth, bytes)
}

func (m *MIDIFile) clicks() Clicks {
	clks := float32(m.currtime) / m.clickfactor
	return (Clicks)(clks + 0.5) // round it
}

func (m *MIDIFile) queuenote(synth string, pitch, velocity byte, notetype string) *Note {
	n := &Note{
		TypeOf:   notetype,
		Clicks:   m.clicks(),
		Pitch:    pitch,
		Velocity: velocity,
		Duration: UnfinishedDuration,
		Synth:    synth,
		next:     nil,
	}
	m.add2noteq(n)
	return n
}

func (m *MIDIFile) add2noteq(n *Note) {
	m.noteq = m.noteq.InsertNoLock(n)
	m.noteq.ResetLengthNoLock()
	m.numq++

}

func (m *MIDIFile) readTrack() error {

	// This array is indexed by the high half of a status byte.  It's
	// value is either the number of bytes needed (1 or 2) for a channel
	// message, or 0 (meaning it's not  a channel message).
	chantype := []int{
		0, 0, 0, 0, 0, 0, 0, 0, // 0x00 through 0x70
		2, 2, 2, 2, 1, 1, 2, 0, // 0x80 through 0xf0
	}

	sysexcontinue := false // if last message was an unfinished sysex
	running := false       // true when running status used
	status := byte(0)      // (possibly running) status byte

	s, err := m.readMT()
	if err != nil {
		return err
	}
	if s != "MTrk" {
		return fmt.Errorf("unexpected string, looking for MTrk, got %s", s)
	}

	m.toberead, err = m.read32bit()
	if err != nil {
		return err
	}
	m.currtime = 0

	m.starttrack()

	for m.toberead > 0 {

		dt, err := m.readvarinum() // delta time
		if err != nil {
			return err
		}
		if dt < 0 {
			return fmt.Errorf("warning: negative delta time (%d) in MIDIFile", dt)
		}
		m.currtime += dt

		c, err := m.readc() // decrements m.toberead - does toberead need to be inside MIDIFile?
		if err != nil {
			return err
		}

		if sysexcontinue && c != 0xf7 {
			Warn("Didn't find expected continuation of a sysex?")
		}

		if (c & 0x80) == 0 { // running status?
			if status == 0 {
				Warn("Unexpected running status")
			}
			running = true
		} else {
			status = c
			running = false
		}

		needed := chantype[(status>>4)&0xf]

		if needed != 0 { // ie. is it a channel message?

			var c1 byte
			if running {
				c1 = c
			} else {
				c1, err = m.readc()
				if err != nil {
					return err
				}
				c1 = c1 & 0x7f
			}

			// The &0xf7 here may seem unnecessary, but I've seen
			// 'bad' midi files that had, e.g., volume bytes
			// with the upper bit set.  This code should not harm
			// proper data.

			if needed == 1 {
				m.chanmessage(status, c1, 0)
			} else if needed == 2 {
				c2, err := m.readc()
				if err != nil {
					return err
				}
				c2 = c2 & 0x7f
				m.chanmessage(status, c1, c2)
			} else {
				return fmt.Errorf("unexpected value for needed: %d", needed)
			}
			continue
		}

		// Non-channel messages end up here
		switch c {
		case 0xff: // meta event

			metatype, err := m.readc()
			if err != nil {
				return err
			}
			// watch out - Don't combine the next 2 statements
			lng, err := m.readvarinum()
			if err != nil {
				return err
			}
			lookfor := m.toberead - lng
			m.msginit()

			for m.toberead > lookfor {
				c, err := m.readc()
				if err != nil {
					return err
				}
				m.msgadd(c)
			}

			m.metaevent(metatype)

		case 0xf0: // start of system exclusive

			// watch out - Don't combine the next 2 statements
			lng, err := m.readvarinum()
			if err != nil {
				return err
			}
			lookfor := m.toberead - lng
			m.msginit()
			m.msgadd(0xf0)

			for m.toberead > lookfor {
				c, err := m.readc()
				if err != nil {
					return err
				}
				m.msgadd(c)
			}

			if c == 0xf7 || m.dosysexcontinue {
				m.sysex("", m.msgbytes())
			} else {
				sysexcontinue = true
			}

		case 0xf7: // sysex continuation or arbitrary stuff

			// watch out - Don't combine the next 2 statements
			lng, err := m.readvarinum()
			if err != nil {
				return err
			}
			lookfor := m.toberead - lng

			if !sysexcontinue {
				m.msginit()
			}

			for m.toberead > lookfor {
				c, err := m.readc()
				if err != nil {
					return err
				}
				m.msgadd(c)
			}

			if !sysexcontinue {
				m.arbitrary("", m.msgbytes())
			} else if c == 0xf7 {
				m.sysex("", m.msgbytes())
				sysexcontinue = false
			}

		default:
			Warn("unexpected midi byte", "byte", c)
		}
	}
	m.endtrack()

	return nil
}

func (m *MIDIFile) readc() (byte, error) {
	if m.sofar < len(m.dat) {
		c := m.dat[m.sofar]
		m.sofar++
		m.toberead--
		return c, nil
	}
	return 0, io.EOF
}

func (m *MIDIFile) read32bit() (int, error) {
	b0, _ := m.readc()
	b1, _ := m.readc()
	b2, _ := m.readc()
	b3, err := m.readc()
	if err != nil {
		return 0, err
	}
	var bytes = []byte{b0, b1, b2, b3}
	value := binary.BigEndian.Uint32(bytes)
	return int(value), nil
}

func (m *MIDIFile) read16bit() (int, error) {
	b0, _ := m.readc()
	b1, err := m.readc()
	if err != nil {
		return 0, err
	}
	var bytes = []byte{b0, b1}
	value := binary.BigEndian.Uint16(bytes)
	return int(value), nil
}

func (m *MIDIFile) readvarinum() (int, error) {
	c, err := m.readc()
	if err != nil {
		return 0, err
	}
	value := int(c)
	if (c & 0x80) != 0 {
		value &= 0x7f
		for {
			c, err := m.readc()
			if err != nil {
				return 0, err
			}
			value = (value << 7) + int(c&0x7f)
			if (c & 0x80) == 0 {
				break
			}
		}
	}
	return value, nil
}

*/

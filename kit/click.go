package kit

import (
	"math"
	"sync"
)

// Clicks is a time or duration value.
// NOTE: A Clicks value can be negative because
// it's sometimes relative to the starting time of a Phrase.
// XXX - possiblycould have a type to distinguish Clicks that are
// XXX - used as absolute time versus Clicks that are step numbers
type Clicks int64

var WholeNote = Clicks(96)
var HalfNote = Clicks(48)
var QuarterNote = Clicks(24)
var EighthNote = Clicks(12)
var SixteenthNote = Clicks(6)
var ThirtySecondNote = Clicks(3)

// MaxClicks is the high-possible value for Clicks
var MaxClicks = Clicks(math.MaxInt64)

// XXX - having Mutexes for all of these values
// is probably silly, should be simplified

var engineCurrentMilli int64
var engineCurrentMilliMutex sync.RWMutex

var engineCurrentMilliOffset int64
var engineCurrentMilliOffsetMutex sync.RWMutex

var engineCurrentClickOffset Clicks
var engineCurrentClickOffsetMutex sync.RWMutex

var engineClicksPerSecond Clicks
var engineClicksPerSecondMutex sync.RWMutex

var engineCurrentClick Clicks
var globaclCurrentClickMutex sync.RWMutex

var OneBeat Clicks
var OneBeatMutex sync.RWMutex

// CurrentMilli is the time from the start, in milliseconds
const defaultClicksPerSecond = Clicks(192)
const minClicksPerSecond = (defaultClicksPerSecond / 16)
const maxClicksPerSecond = (defaultClicksPerSecond * 16)

// var loopForever = 999999

// Bits for Events
const EventMidiInput = 0x01
const EventNoteOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventNoteOutput | EventCursor

type ClickEvent struct {
	Click  Clicks
	Uptime float64
}

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec Clicks) {
	// no locks needed here
	engineClicksPerSecond = clkpersec
	engineCurrentMilliOffset = 0
	engineCurrentClickOffset = 0
	OneBeat = Clicks(engineClicksPerSecond / 2) // i.e. 120bpm
}

// ChangeClicksPerSecond is what you use to change the tempo
func ChangeClicksPerSecond(factor float64) {
	TempoFactor = factor
	cpsf := float64(defaultClicksPerSecond)
	clkpersec := Clicks(cpsf * factor)
	if clkpersec < minClicksPerSecond {
		clkpersec = minClicksPerSecond
	}
	if clkpersec > maxClicksPerSecond {
		clkpersec = maxClicksPerSecond
	}

	SetCurrentMilliOffset(CurrentMilli())
	SetCurrentClickOffset(CurrentClick())
	SetClicksPerSecond(clkpersec)

	OneBeat = Clicks(ClicksPerSecond() / 2)
}

// Seconds2Clicks converts a Time value (elapsed seconds) to Clicks
func Seconds2Clicks(tm float64) Clicks {
	clickOffset := CurrentClickOffset()
	cps := ClicksPerSecond()
	milliOffset := CurrentMilliOffset()
	cpsf := float64(cps)
	var tmpclicks = Clicks(0.5 + float64(tm*1000-float64(milliOffset))*(cpsf/1000.0))
	click := clickOffset + tmpclicks
	return click
}

// Clicks2Seconds converts Clicks to Time (seconds), relative
func Clicks2Seconds(clk Clicks) float64 {
	cps := engineClicksPerSecond
	clkf := float64(clk)
	return clkf / float64(cps)
}

// TempoFactor xxx
var TempoFactor = float64(1.0)

// CurrentMilli

func CurrentMilli() int64 {
	engineCurrentMilliMutex.RLock()
	defer engineCurrentMilliMutex.RUnlock()
	return engineCurrentMilli
}

func SetCurrentMilli(milli int64) {
	engineCurrentMilliMutex.Lock()
	engineCurrentMilli = milli
	engineCurrentMilliMutex.Unlock()
}

//  CurrentMilliOffset

func CurrentMilliOffset() int64 {
	engineCurrentMilliOffsetMutex.RLock()
	defer engineCurrentMilliOffsetMutex.RUnlock()
	return engineCurrentMilliOffset
}

func SetCurrentMilliOffset(milli int64) {
	engineCurrentMilliOffsetMutex.Lock()
	engineCurrentMilliOffset = milli
	engineCurrentMilliOffsetMutex.Unlock()
}

//  ClicksPerSecond

func ClicksPerSecond() Clicks {
	engineClicksPerSecondMutex.RLock()
	defer engineClicksPerSecondMutex.RUnlock()
	return Clicks(engineClicksPerSecond)
}

func SetClicksPerSecond(cps Clicks) {
	engineClicksPerSecondMutex.Lock()
	engineClicksPerSecond = cps
	engineClicksPerSecondMutex.Unlock()
}

// CurrentClick

func CurrentClick() Clicks {
	globaclCurrentClickMutex.RLock()
	defer globaclCurrentClickMutex.RUnlock()
	return engineCurrentClick
}

func SetCurrentClick(click Clicks) {
	globaclCurrentClickMutex.Lock()
	engineCurrentClick = click
	globaclCurrentClickMutex.Unlock()
}

// CurrentClickOffset

func CurrentClickOffset() Clicks {
	engineCurrentClickOffsetMutex.RLock()
	defer engineCurrentClickOffsetMutex.RUnlock()
	return engineCurrentClickOffset
}

func SetCurrentClickOffset(click Clicks) {
	engineCurrentClickOffsetMutex.Lock()
	engineCurrentClickOffset = click
	engineCurrentClickOffsetMutex.Unlock()
}

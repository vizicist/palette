package engine

import (
	"math"
	"strconv"
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

var globalCurrentMilli int64
var globalCurrentMilliMutex sync.RWMutex

var globalCurrentMilliOffset int64
var globalCurrentMilliOffsetMutex sync.RWMutex

var globalCurrentClickOffset Clicks
var globalCurrentClickOffsetMutex sync.RWMutex

var globalClicksPerSecond Clicks
var globalClicksPerSecondMutex sync.RWMutex

var globalCurrentClick Clicks
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

func (ce ClickEvent) ToMap() map[string]string {
	return map[string]string{
		"event":  "click",
		"click":  strconv.FormatInt(int64(ce.Click), 10),
		"uptime": strconv.FormatFloat(ce.Uptime, 'f', 6, 64),
	}
}

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec Clicks) {
	// no locks needed here
	globalClicksPerSecond = clkpersec
	globalCurrentMilliOffset = 0
	globalCurrentClickOffset = 0
	OneBeat = Clicks(globalClicksPerSecond / 2) // i.e. 120bpm
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
	cps := globalClicksPerSecond
	clkf := float64(clk)
	return clkf / float64(cps)
}

// TempoFactor xxx
var TempoFactor = float64(1.0)

// CurrentMilli

func CurrentMilli() int64 {
	globalCurrentMilliMutex.RLock()
	defer globalCurrentMilliMutex.RUnlock()
	return globalCurrentMilli
}

func SetCurrentMilli(milli int64) {
	globalCurrentMilliMutex.Lock()
	globalCurrentMilli = milli
	globalCurrentMilliMutex.Unlock()
}

//  CurrentMilliOffset

func CurrentMilliOffset() int64 {
	globalCurrentMilliOffsetMutex.RLock()
	defer globalCurrentMilliOffsetMutex.RUnlock()
	return globalCurrentMilliOffset
}

func SetCurrentMilliOffset(milli int64) {
	globalCurrentMilliOffsetMutex.Lock()
	globalCurrentMilliOffset = milli
	globalCurrentMilliOffsetMutex.Unlock()
}

//  ClicksPerSecond

func ClicksPerSecond() Clicks {
	globalClicksPerSecondMutex.RLock()
	defer globalClicksPerSecondMutex.RUnlock()
	return Clicks(globalClicksPerSecond)
}

func SetClicksPerSecond(cps Clicks) {
	globalClicksPerSecondMutex.Lock()
	globalClicksPerSecond = cps
	globalClicksPerSecondMutex.Unlock()
}

// CurrentClick

func CurrentClick() Clicks {
	globaclCurrentClickMutex.RLock()
	defer globaclCurrentClickMutex.RUnlock()
	return globalCurrentClick
}

func SetCurrentClick(click Clicks) {
	globaclCurrentClickMutex.Lock()
	globalCurrentClick = click
	globaclCurrentClickMutex.Unlock()
}

// CurrentClickOffset

func CurrentClickOffset() Clicks {
	globalCurrentClickOffsetMutex.RLock()
	defer globalCurrentClickOffsetMutex.RUnlock()
	return globalCurrentClickOffset
}

func SetCurrentClickOffset(click Clicks) {
	globalCurrentClickOffsetMutex.Lock()
	globalCurrentClickOffset = click
	globalCurrentClickOffsetMutex.Unlock()
}

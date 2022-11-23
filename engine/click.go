package engine

import (
	"sync"
)

// XXX - having Mutexes for all of these values
// is probably silly, should be simplified

var globalCurrentMilli int64
var globalCurrentMilliMutex sync.RWMutex

var globalCurrentMilliOffset int64
var globalCurrentMilliOffsetMutex sync.RWMutex

var globalCurrentClickOffset Clicks
var globalCurrentClickOffsetMutex sync.RWMutex

var globalClicksPerSecond int
var globalClicksPerSecondMutex sync.RWMutex

var globalCurrentClick Clicks
var globaclCurrentClickMutex sync.RWMutex

var OneBeat Clicks
var OneBeatMutex sync.RWMutex

// CurrentMilli is the time from the start, in milliseconds
const defaultClicksPerSecond = 192
const minClicksPerSecond = (defaultClicksPerSecond / 16)
const maxClicksPerSecond = (defaultClicksPerSecond * 16)

var defaultSynth = "default"

// var loopForever = 999999

// Bits for Events
const EventMidiInput = 0x01
const EventNoteOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventNoteOutput | EventCursor

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec int) {
	// no locks needed here
	globalClicksPerSecond = clkpersec
	globalCurrentMilliOffset = 0
	globalCurrentClickOffset = 0
	OneBeat = Clicks(globalClicksPerSecond / 2) // i.e. 120bpm
}

// ChangeClicksPerSecond is what you use to change the tempo
func ChangeClicksPerSecond(factor float64) {
	TempoFactor = factor
	clkpersec := int(defaultClicksPerSecond * factor)
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
	click := clickOffset + Clicks(0.5+float64(tm*1000-float64(milliOffset))*(float64(cps)/1000.0))
	return click
}

// Clicks2Seconds converts Clicks to Time (seconds), relative
func Clicks2Seconds(clk Clicks) float64 {
	return float64(clk) / float64(globalClicksPerSecond)
}

/*
// Clicks2Seconds converts Clicks to Time (seconds), absolute
func Clicks2SecondsAbsolute(clk Clicks) float64 {
	// Take current*Offset values into account
	clk -= currentClickOffset
	secs := float64(clk) / float64(clicksPerSecond)
	secs -= (float64(currentMilliOffset) * 1000.0)
	return secs
}
*/

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

func ClicksPerSecond() int {
	globalClicksPerSecondMutex.RLock()
	defer globalClicksPerSecondMutex.RUnlock()
	return globalClicksPerSecond
}

func SetClicksPerSecond(cps int) {
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

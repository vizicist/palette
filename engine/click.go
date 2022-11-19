package engine

import (
	"sync"
)

var GlobalCurrentMilli int64
var GlobalCurrentMilliMutex sync.Mutex

var GlobalCurrentMilliOffset int64
var GlobalCurrentClickOffset Clicks
var GlobalClicksPerSecond int
var OneBeat Clicks

var GlobalCurrentClick Clicks
var GlobaclCurrentClickMutex sync.RWMutex

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

func CurrentClick() Clicks {
	GlobaclCurrentClickMutex.Lock()
	defer GlobaclCurrentClickMutex.Unlock()
	return GlobalCurrentClick
}

func SetCurrentClick(clk Clicks) {
	GlobaclCurrentClickMutex.Lock()
	GlobalCurrentClick = clk
	GlobaclCurrentClickMutex.Unlock()
}

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec int) {
	GlobalClicksPerSecond = clkpersec
	GlobalCurrentMilliOffset = 0
	GlobalCurrentClickOffset = 0
	OneBeat = Clicks(GlobalClicksPerSecond / 2) // i.e. 120bpm
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
	GlobalCurrentMilliOffset = CurrentMilli()
	GlobalCurrentClickOffset = CurrentClick()
	GlobalClicksPerSecond = clkpersec
	OneBeat = Clicks(GlobalClicksPerSecond / 2)
}

// Seconds2Clicks converts a Time value (elapsed seconds) to Clicks
func Seconds2Clicks(tm float64) Clicks {
	return GlobalCurrentClickOffset + Clicks(0.5+float64(tm*1000-float64(GlobalCurrentMilliOffset))*(float64(GlobalClicksPerSecond)/1000.0))
}

// Clicks2Seconds converts Clicks to Time (seconds), relative
func Clicks2Seconds(clk Clicks) float64 {
	return float64(clk) / float64(GlobalClicksPerSecond)
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

func CurrentMilli() int64 {
	GlobalCurrentMilliMutex.Lock()
	defer GlobalCurrentMilliMutex.Unlock()
	return int64(GlobalCurrentMilli)
}

func SetCurrentMilli(m int64) {
	GlobalCurrentMilliMutex.Lock()
	GlobalCurrentMilli = m
	GlobalCurrentMilliMutex.Unlock()
}

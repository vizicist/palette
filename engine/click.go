package engine

import (
	"sync"
)

var currentMilli int64
var currentMilliMutex sync.Mutex

var currentMilliOffset int64
var currentClickOffset Clicks
var clicksPerSecond int
var oneBeat Clicks

var currentClick Clicks
var currentClickMutex sync.Mutex

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
	currentClickMutex.Lock()
	defer currentClickMutex.Unlock()
	return currentClick
}

func SetCurrentClick(clk Clicks) {
	currentClickMutex.Lock()
	currentClick = clk
	currentClickMutex.Unlock()
}

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec int) {
	clicksPerSecond = clkpersec
	currentMilliOffset = 0
	currentClickOffset = 0
	oneBeat = Clicks(clicksPerSecond / 2) // i.e. 120bpm
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
	currentMilliOffset = CurrentMilli()
	currentClickOffset = CurrentClick()
	clicksPerSecond = clkpersec
	oneBeat = Clicks(clicksPerSecond / 2)
}

// Seconds2Clicks converts a Time value (elapsed seconds) to Clicks
func Seconds2Clicks(tm float64) Clicks {
	return currentClickOffset + Clicks(0.5+float64(tm*1000-float64(currentMilliOffset))*(float64(clicksPerSecond)/1000.0))
}

// Clicks2Seconds converts Clicks to Time (seconds), relative
func Clicks2Seconds(clk Clicks) float64 {
	return float64(clk) / float64(clicksPerSecond)
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
	currentMilliMutex.Lock()
	defer currentMilliMutex.Unlock()
	return int64(currentMilli)
}

func SetCurrentMilli(m int64) {
	currentMilliMutex.Lock()
	currentMilli = m
	currentMilliMutex.Unlock()
}

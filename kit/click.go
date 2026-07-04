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

// engineTiming holds all engine clock state under a single lock, so related
// values (e.g. the offsets used by Seconds2Clicks) are read consistently.
type engineTiming struct {
	mutex           sync.RWMutex
	milli           int64
	milliOffset     int64
	clickOffset     Clicks
	clicksPerSecond Clicks
	click           Clicks
}

var theTiming engineTiming

// OneBeat is written only at initialization and on tempo changes
// (InitializeClicksPerSecond / ChangeClicksPerSecond) and read everywhere.
var OneBeat Clicks

const defaultClicksPerSecond = Clicks(192)
const minClicksPerSecond = (defaultClicksPerSecond / 16)
const maxClicksPerSecond = (defaultClicksPerSecond * 16)

// var loopForever = 999999

const EventMidiInput = 0x01
const EventNoteOutput = 0x02
const EventCursor = 0x04
const EventAll = EventMidiInput | EventNoteOutput | EventCursor

type ClickEvent struct {
	Click  Clicks
	Uptime float64
}

func InitializeClicks() {
	InitializeClicksPerSecond(defaultClicksPerSecond)
}

// InitializeClicksPerSecond initializes
func InitializeClicksPerSecond(clkpersec Clicks) {
	theTiming.mutex.Lock()
	theTiming.clicksPerSecond = clkpersec
	theTiming.milliOffset = 0
	theTiming.clickOffset = 0
	theTiming.mutex.Unlock()
	OneBeat = Clicks(clkpersec / 2) // i.e. 120bpm
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

	theTiming.mutex.Lock()
	theTiming.milliOffset = theTiming.milli
	theTiming.clickOffset = theTiming.click
	theTiming.clicksPerSecond = clkpersec
	theTiming.mutex.Unlock()

	OneBeat = Clicks(clkpersec / 2)
}

// Seconds2Clicks converts a Time value (elapsed seconds) to Clicks
func Seconds2Clicks(tm float64) Clicks {
	theTiming.mutex.RLock()
	clickOffset := theTiming.clickOffset
	cps := theTiming.clicksPerSecond
	milliOffset := theTiming.milliOffset
	theTiming.mutex.RUnlock()

	cpsf := float64(cps)
	var tmpclicks = Clicks(0.5 + float64(tm*1000-float64(milliOffset))*(cpsf/1000.0))
	return clickOffset + tmpclicks
}

// Clicks2Seconds converts Clicks to Time (seconds), relative
func Clicks2Seconds(clk Clicks) float64 {
	return float64(clk) / float64(ClicksPerSecond())
}

// TempoFactor xxx
var TempoFactor = float64(1.0)

func CurrentMilli() int64 {
	theTiming.mutex.RLock()
	defer theTiming.mutex.RUnlock()
	return theTiming.milli
}

func SetCurrentMilli(milli int64) {
	theTiming.mutex.Lock()
	theTiming.milli = milli
	theTiming.mutex.Unlock()
}

func CurrentMilliOffset() int64 {
	theTiming.mutex.RLock()
	defer theTiming.mutex.RUnlock()
	return theTiming.milliOffset
}

func SetCurrentMilliOffset(milli int64) {
	theTiming.mutex.Lock()
	theTiming.milliOffset = milli
	theTiming.mutex.Unlock()
}

func ClicksPerSecond() Clicks {
	theTiming.mutex.RLock()
	defer theTiming.mutex.RUnlock()
	return theTiming.clicksPerSecond
}

func SetClicksPerSecond(cps Clicks) {
	theTiming.mutex.Lock()
	theTiming.clicksPerSecond = cps
	theTiming.mutex.Unlock()
}

func CurrentClick() Clicks {
	theTiming.mutex.RLock()
	defer theTiming.mutex.RUnlock()
	return theTiming.click
}

func SetCurrentClick(click Clicks) {
	theTiming.mutex.Lock()
	theTiming.click = click
	theTiming.mutex.Unlock()
}

func CurrentClickOffset() Clicks {
	theTiming.mutex.RLock()
	defer theTiming.mutex.RUnlock()
	return theTiming.clickOffset
}

func SetCurrentClickOffset(click Clicks) {
	theTiming.mutex.Lock()
	theTiming.clickOffset = click
	theTiming.mutex.Unlock()
}

package gui

import (
	"fmt"
	"log"
	"strings"
)

// CurrentWindName is the name of the active page.
var CurrentWindName string

/*
var red = nanovgo.RGBA(255, 0, 0, 255)
var black = nanovgo.RGBA(0, 0, 0, 255)
var white = nanovgo.RGBA(255, 255, 255, 255)

// Style xxx
type Style struct {
	fontSize    float32
	fontFace    string
	textColor   nanovgo.Color
	strokeColor nanovgo.Color
	fillColor   nanovgo.Color
	charWidth   int
	lineHeight  int
}

// DefaultStyle is for initializing Style values
var DefaultStyle Style = Style{
	fontSize:    12.0,
	fontFace:    "lucida",
	textColor:   black,
	strokeColor: black,
	fillColor:   white,
	charWidth:   0, // filled in by SetSize
	lineHeight:  0, // filled in by SetSize
}

// Do xxx
func (style Style) Do(ctx *nanovgo.Context) {
	ctx.SetFillColor(style.fillColor)
	ctx.SetStrokeColor(style.strokeColor)
	ctx.SetFontFace(style.fontFace)
	ctx.SetFontSize(style.fontSize)
}
*/

// IconSEARCH, etc
const (
	IconDOWN  = 0x25BE
	IconUP    = 0x25B4
	IconLEFT  = 0x25C2
	IconRIGHT = 0x25B8
	// IconSEARCH       = 0x1F50D
	// IconCIRCLEDCROSS = 0x2716
	// IconCHEVRONRIGHT = 0xE75E
	// IconCHECK        = 0x2713
	// IconLOGIN        = 0xE740
	// IconTRASH        = 0xE729
)

/*
// SwitchToWind xxx
func SwitchToWind(name string) {
	wind, ok := Wind[name]
	if !ok {
		log.Printf("No page named: %s\n", name)
	} else {
		CurrentWindName = name
		wind.SetFocus(nil)
	}
}
*/

func (wind *VizObjData) venueAPI(venue string, api string) {
	log.Printf(fmt.Sprintf("venueAPI venue=%s, api=%s\n", venue, api))
}

func (wind *VizObjData) addStatusButtons(x, y int) {
	/*
		bh := int(2.5 * float32(wind.style.lineHeight))
		bw := 12 * wind.style.charWidth

		venue := "PhotonSalon1"

			b := NewButton("startvenue", "Start\nVenue", x, y, bw, bh, wind.style,
				func(text string) { wind.venueAPI(venue, "start") })
			wind.AddObject(b)

			x += bw + wind.style.charWidth

			b = NewButton("stopvenue", "Stop\nVenue", x, y, bw, bh, wind.style,
				func(text string) { wind.venueAPI(venue, "stop") })
			wind.AddObject(b)

			x += bw + wind.style.charWidth

			b = NewButton("recordon", "Record\nOn", x, y, bw, bh, wind.style,
				func(name string) { wind.venueAPI(venue, "recording.start") })
			wind.AddObject(b)

			x += bw + wind.style.charWidth

			b = NewButton("recordoff", "Record\nOff", x, y, bw, bh, wind.style,
				func(name string) { wind.venueAPI(venue, "recording.stop") })
			wind.AddObject(b)

			x += bw + wind.style.charWidth

			b = NewButton("recordplayback", "Record\nPlayback", x, y, bw, bh, wind.style,
				func(name string) { wind.venueAPI(venue, "recording.playback") })
			wind.AddObject(b)

			x += bw + wind.style.charWidth

			b = NewButton("playmidifile", "Play\nMIDIFile", x, y, bw, bh, wind.style,
				func(name string) {
					log.Printf("playmidifile button name=%s\n", name)
				})
			wind.AddObject(b)
	*/
}

// var logLines []string
var logLines = 6
var logTexts []*VizText

// VizLog xxx
func VizLog(s string) {
	for n := 1; n < logLines; n++ {
		logTexts[n-1].text = logTexts[n].text
	}
	// Remove anything after a newline
	if newline := strings.Index(s, "\n"); newline >= 0 {
		s = s[:newline]
	}
	logTexts[logLines-1].text = s
}

func (wind *VizObjData) addSettings(x, y int) {

	/*
		labelw := wind.Rect().Dx() / 4
		valuew := wind.Rect().Dx() / 2
		h := wind.Style.lineHeight

		midiCombo := NewCombo("midiinput", "MIDI Input",
			x, y, labelw, valuew, h, wind.Style, wind.localsettingCallback)
		midiCombo.addValue("microKEY2 Air")
		midiCombo.addValue("01. Internal MIDI")
		midiCombo.addValue("02. Internal MIDI")
		midiCombo.addValue("03. Internal MIDI")
		midiCombo.addValue("04. Internal MIDI")
		wind.AddObject(midiCombo)

		y += wind.Style.lineHeight

		midiFileCombo := NewCombo("midifile", "MIDI File",
			x, y, labelw, valuew, h, wind.Style, wind.localsettingCallback)

		log.Printf("")
		venue := "PhotonSalon1"
		midifiles, err := engine.VenueMidifiles(venue)
		if err != nil {
			log.Printf("LoadVenue: VenueMidifiles err=%s\n", err)
		} else {
			for _, nm := range midifiles {
				midiFileCombo.addValue(nm)
			}
		}
		wind.AddObject(midiFileCombo)
	*/
}

func (wind *VizObjData) addLogArea(nloglines int, x, y int) {
	/*

		w := wind.Rect().Dx()
		h := wind.Style.lineHeight

		t := NewText("Message Log:", image.Rect(x, y, x+w-10*wind.Style.charWidth, y+h), wind.Style))
		wind.AddObject("log", NewText("Message Log:", image.Rect(x, y, x+w-10*wind.Style.charWidth, y+h), wind.Style))
		y += wind.Style.lineHeight
		// It might be better to re-use
		// the Texts if they've already been created.
		newlogTexts := make([]*VizText, 0)
		for n := 0; n < nloglines; n++ {
			// if there's existing logTexts, use their text
			var t string
			if logTexts != nil {
				t = logTexts[n].text
			}
			nm := fmt.Sprintf("line%d", n)
			txt := NewText(t, image.Rect(x, y, x+w, y+h), wind.Style)
			newlogTexts = append(newlogTexts, txt)
			wind.AddObject(nm, txt)
			y += wind.Style.lineHeight
		}
		logTexts = newlogTexts
		return wind
	*/
}

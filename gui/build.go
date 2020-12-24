package gui

import (
	"fmt"
	"log"
	"strings"

	"github.com/micaelAlastor/nanovgo"
	"github.com/vizicist/palette/engine"
)

// Page xxx
var Page map[string]*VizPage = make(map[string]*VizPage)

// CurrentPageName is the name of the active page.
// Since Pages get rebuilt if the window size changes,
// we keep track of the name, not the VizPage
var CurrentPageName string

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
	charWidth   float32
	lineHeight  float32
}

func (pg *VizPage) defaultStyle() Style {
	s := Style{
		fontSize:    18.0,
		fontFace:    "lucida",
		textColor:   black,
		strokeColor: black,
		fillColor:   white,
		lineHeight:  pg.height / 48.0,
		charWidth:   pg.width / 80.0,
	}
	return s
}

// Do xxx
func (style Style) Do(ctx *nanovgo.Context) {
	ctx.SetFillColor(style.fillColor)
	ctx.SetStrokeColor(style.strokeColor)
	ctx.SetFontFace(style.fontFace)
	ctx.SetFontSize(style.fontSize)
}

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

// SwitchToPage xxx
func SwitchToPage(name string) {
	pg, ok := Page[name]
	if !ok {
		log.Printf("No page named: %s\n", name)
	} else {
		CurrentPageName = name
		pg.SetFocus(nil)
	}
}

func (pg *VizPage) addPageHeader() {
	x := float32(10.0)
	y := float32(10.0)
	bh := float32(1.5) * pg.style.lineHeight
	bw := 12 * pg.style.charWidth

	b := NewButton("status", "Status", x, y, bw, bh, pg.style,
		func(updown string) {
			if updown == "down" {
				SwitchToPage("status")
			}
		})

	// b.SetWaitForUp(true)
	pg.AddObject(b)

	x += bw + pg.style.charWidth

	b = NewButton("misc", "Misc", x, y, bw, bh, pg.style,
		func(updown string) {
			if updown == "down" {
				SwitchToPage("misc")
			}
		})
	// b.SetWaitForUp(true)
	pg.AddObject(b)

	x += bw + pg.style.charWidth

	b = NewButton("venues", "Venues", x, y, bw, bh, pg.style,
		func(updown string) {
			if updown == "down" {
				SwitchToPage("venues")
			}
		})
	// b.SetWaitForUp(true)
	pg.AddObject(b)

}

func (pg *VizPage) venueAPI(venue string, api string) {
	log.Printf(fmt.Sprintf("venueAPI venue=%s, api=%s\n", venue, api))
}

func (pg *VizPage) addStatusButtons(x, y float32) {
	bh := float32(2.5) * pg.style.lineHeight
	bw := 12 * pg.style.charWidth

	venue := "PhotonSalon1"

	b := NewButton("startvenue", "Start\nVenue", x, y, bw, bh, pg.style,
		func(text string) { pg.venueAPI(venue, "start") })
	pg.AddObject(b)

	x += bw + pg.style.charWidth

	b = NewButton("stopvenue", "Stop\nVenue", x, y, bw, bh, pg.style,
		func(text string) { pg.venueAPI(venue, "stop") })
	pg.AddObject(b)

	x += bw + pg.style.charWidth

	b = NewButton("recordon", "Record\nOn", x, y, bw, bh, pg.style,
		func(name string) { pg.venueAPI(venue, "recording.start") })
	pg.AddObject(b)

	x += bw + pg.style.charWidth

	b = NewButton("recordoff", "Record\nOff", x, y, bw, bh, pg.style,
		func(name string) { pg.venueAPI(venue, "recording.stop") })
	pg.AddObject(b)

	x += bw + pg.style.charWidth

	b = NewButton("recordplayback", "Record\nPlayback", x, y, bw, bh, pg.style,
		func(name string) { pg.venueAPI(venue, "recording.playback") })
	pg.AddObject(b)

	x += bw + pg.style.charWidth

	b = NewButton("playmidifile", "Play\nMIDIFile", x, y, bw, bh, pg.style,
		func(name string) {
			log.Printf("playmidifile button name=%s\n", name)
		})
	pg.AddObject(b)
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

// StatusPage xxx
func StatusPage(ctx *nanovgo.Context, width, height float32) *VizPage {

	pg := NewPage(ctx, width, height)
	pg.addPageHeader()

	nloglines := 6
	x := float32(pg.style.charWidth)
	y := float32(height - float32(nloglines+2)*pg.style.lineHeight)
	pg.addLogArea(nloglines, x, y)

	x = float32(pg.style.charWidth)
	y = float32(0.4 * height)
	pg.addSettings(x, y)

	x = float32(pg.style.charWidth)
	y = float32(0.2 * height)
	pg.addStatusButtons(x, y)

	return pg
}

func (pg *VizPage) localsettingCallback(c *VizCombo, choice int) {
	c.choice = choice
	val := c.choices[c.choice]
	pg.localSettings[c.Name()] = val
	log.Printf("localsettingCallback choice=%d\n", choice)
}

func (pg *VizPage) addSettings(x, y float32) {

	labelw := pg.width / 4.0
	valuew := pg.width / 2.0
	h := pg.style.lineHeight

	midiCombo := NewCombo("midiinput", "MIDI Input",
		x, y, labelw, valuew, h, pg.style, pg.localsettingCallback)
	midiCombo.addValue("microKEY2 Air")
	midiCombo.addValue("01. Internal MIDI")
	midiCombo.addValue("02. Internal MIDI")
	midiCombo.addValue("03. Internal MIDI")
	midiCombo.addValue("04. Internal MIDI")
	pg.AddObject(midiCombo)

	y += pg.style.lineHeight

	midiFileCombo := NewCombo("midifile", "MIDI File",
		x, y, labelw, valuew, h, pg.style, pg.localsettingCallback)

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
	pg.AddObject(midiFileCombo)
}

func (pg *VizPage) addLogArea(nloglines int, x, y float32) *VizPage {

	w := pg.width
	h := pg.style.lineHeight

	pg.AddObject(NewText("log", "Message Log:", x, y, w-10*pg.style.charWidth, h, pg.style))
	y += pg.style.lineHeight
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
		txt := NewText(nm, t, x, y, w, h, pg.style)
		newlogTexts = append(newlogTexts, txt)
		pg.AddObject(txt)
		y += pg.style.lineHeight
	}
	logTexts = newlogTexts
	return pg
}

// MiscPage xxx
func MiscPage(ctx *nanovgo.Context, width, height float32) *VizPage {
	pg := NewPage(ctx, width, height)
	pg.addPageHeader()
	x := float32(10.0)
	y := float32(height - 200)
	pg.AddObject(NewText("misc", "This is the Misc Page", x, y, 100, 100, pg.style))
	return pg
}

// VenuesPage xxx
func VenuesPage(ctx *nanovgo.Context, width, height float32) *VizPage {
	pg := NewPage(ctx, width, height)
	pg.addPageHeader()
	x := float32(10.0)
	y := float32(height - 200)
	pg.AddObject(NewText("misc", "This is the Venues Page", x, y, 100, 100, pg.style))
	return pg
}

// BuildPages xxx
func BuildPages(ctx *nanovgo.Context, width, height float32) {
	Page["status"] = StatusPage(ctx, width, height)
	Page["misc"] = MiscPage(ctx, width, height)
	Page["venues"] = VenuesPage(ctx, width, height)
	SwitchToPage("status")
}

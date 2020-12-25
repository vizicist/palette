package gui

import (
	"fmt"
	"image"
	"log"
	"strings"

	"github.com/micaelAlastor/nanovgo"
	"github.com/vizicist/palette/engine"
)

// Wind xxx
// var Wind map[string]*VizWind = make(map[string]*VizWind)

// CurrentWindName is the name of the active page.
var CurrentWindName string

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

func (wind *VizWind) defaultStyle() Style {
	s := Style{
		fontSize:    18.0,
		fontFace:    "lucida",
		textColor:   black,
		strokeColor: black,
		fillColor:   white,
		lineHeight:  int(wind.Rect().Dy() / 48.0),
		charWidth:   int(wind.Rect().Dx() / 80.0),
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

func (wind *VizWind) addHeaderButtons() {

	x0 := wind.rect.Min.X + 10
	y0 := wind.rect.Min.Y + 10

	b := wind.NewButton("status", "Status12345", image.Point{X: x0, Y: y0}, wind.Style,
		func(updown string) {
			if updown == "down" {
				log.Printf("Status button was pressed\n")
				// SwitchToWind("status")
			}
		})

	wind.AddObject(b)

	/*
		buttonDx := 14 * wind.style.charWidth
		x0 += buttonDx
		x1 += buttonDx

		b = NewButton("misc", "Misc", image.Rect(x0, y0, x1, y1), wind.style,
			func(updown string) {
				if updown == "down" {
					SwitchToWind("misc")
				}
			})
		wind.AddObject(b)

		x0 += buttonDx
		x1 += buttonDx

		b = NewButton("venues", "Venues", image.Rect(x0, y0, x1, y1), wind.style,
			func(updown string) {
				if updown == "down" {
					SwitchToWind("venues")
				}
			})
		wind.AddObject(b)
	*/

}

func (wind *VizWind) venueAPI(venue string, api string) {
	log.Printf(fmt.Sprintf("venueAPI venue=%s, api=%s\n", venue, api))
}

func (wind *VizWind) addStatusButtons(x, y int) {
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

// BuildStatusWind xxx
func BuildStatusWind(wind *VizWind) {

	wind.addHeaderButtons()

	/*
		nloglines := 6
		x := wind.style.charWidth
		y := wind.Height()/2 - (nloglines+2)*wind.style.lineHeight
		wind.addLogArea(nloglines, x, y)
	*/

	/*
		x = wind.style.charWidth
		y = int(0.4 * float32(wind.Height()))
		wind.addSettings(x, y)

		x = wind.style.charWidth
		y = int(0.2 * float32(wind.Height()))
		wind.addStatusButtons(x, y)
	*/
}

func (wind *VizWind) localsettingCallback(c *VizCombo, choice int) {
	c.choice = choice
	val := c.choices[c.choice]
	wind.localSettings[c.Name()] = val
	log.Printf("localsettingCallback choice=%d\n", choice)
}

func (wind *VizWind) addSettings(x, y int) {

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
}

func (wind *VizWind) addLogArea(nloglines int, x, y int) *VizWind {

	w := wind.Rect().Dx()
	h := wind.Style.lineHeight

	wind.AddObject(NewText("log", "Message Log:", image.Rect(x, y, x+w-10*wind.Style.charWidth, y+h), wind.Style))
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
		txt := NewText(nm, t, image.Rect(x, y, x+w, y+h), wind.Style)
		newlogTexts = append(newlogTexts, txt)
		wind.AddObject(txt)
		y += wind.Style.lineHeight
	}
	logTexts = newlogTexts
	return wind
}

// BuildMiscWind xxx
func BuildMiscWind(wind *VizWind, rect image.Rectangle) *VizWind {
	// wind.addWindHeader()
	x := rect.Min.X + 200
	y := rect.Min.Y + 200
	wind.AddObject(NewText("misc", "This is the Misc Wind", image.Rect(x, y, x+200, y+200), wind.Style))
	return wind
}

// BuildVenuesWind xxx
func BuildVenuesWind(wind *VizWind, rect image.Rectangle) *VizWind {
	wind.addHeaderButtons()
	x := rect.Min.X + 200
	y := rect.Max.Y - 200
	wind.AddObject(NewText("misc", "This is the Venues Wind", image.Rect(x, y, x+200, y+200), wind.Style))
	return wind
}

// BuildInitialScreen xxx
func BuildInitialScreen(screen *VizScreen) error {

	halfheight := screen.height / 2

	srect := image.Rect(100, 100, screen.width-100, halfheight-20)
	w, err := screen.AddWind("status", srect)
	if err != nil {
		return err
	}

	BuildStatusWind(w)

	srect = image.Rect(100, halfheight, screen.width-100, screen.height-100)
	w, err = screen.AddWind("status2", srect)
	if err != nil {
		return err
	}

	BuildStatusWind(w)

	// Wind["venues"] = VenuesWind(ctx, rect)
	//
	// mrect := image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
	// Wind["misc"] = MiscWind(ctx, mrect)

	// SwitchToWind("status")

	return nil
}

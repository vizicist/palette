package gui

// CurrentWindName is the name of the active page.
var CurrentWindName string

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
func (wind *WindowData) venueAPI(venue string, api string) {
	log.Printf(fmt.Sprintf("venueAPI venue=%s, api=%s\n", venue, api))
}

var logLines = 6
var logTexts []*Text

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

func (wind *WindowData) addSettings(x, y int) {

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

/*
func (wind *WindowData) addLogArea(nloglines int, x, y int) {

		w := wind.Rect().Dx()
		h := wind.Style.lineHeight

		t := NewText("Message Log:", image.Rect(x, y, x+w-10*wind.Style.charWidth, y+h), wind.Style))
		wind.AddObject("log", NewText("Message Log:", image.Rect(x, y, x+w-10*wind.Style.charWidth, y+h), wind.Style))
		y += wind.Style.lineHeight
		// It might be better to re-use
		// the Texts if they've already been created.
		newlogTexts := make([]*Text, 0)
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
}
*/

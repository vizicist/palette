package parse

// Pre-defined macros.  It is REQUIRED that these values match the
// corresponding values in phrase.go and elsewhere  For example, the value
// of P_STORE must match STORE, NT_NOTE must match NOTE, etc.

type Macro struct {
	name     string
	template string
	params   []string
}

var Macros = map[string]Macro{}

func NewMacro(name string, value string, params []string) Macro {
	return Macro{
		name:     name,
		template: value,
		params:   params,
	}
}

func InitMacros() {
	for name, value := range MacrosBuiltin {
		m := Macro{
			name:     name,
			params:   []string{},
			template: value,
		}
		Macros[name] = m
	}
}

var MacrosBuiltin map[string]string = map[string]string{

	/* These are values for nt.type, also used as bit-vals for  */
	/* the value of Filter. */
	"MIDIBYTES":     "1", /* NT_LE3BYTES is not here - not user-visible */
	"NOTE":          "2",
	"NOTEON":        "4",
	"NOTEOFF":       "8",
	"CHANPRESSURE":  "16",
	"CONTROLLER":    "32",
	"PROGRAM":       "64",
	"PRESSURE":      "128",
	"PITCHBEND":     "256",
	"SYSEX":         "512",
	"POSITION":      "1024",
	"CLOCK":         "2048",
	"SONG":          "4096",
	"STARTSTOPCONT": "8192",
	"SYSEXTEXT":     "16384",

	"Nullstr": "\"\"",

	/* Values for action() types.  The values are intended to not */
	/* overlap the values for interrupt(), to avoid misuse and */
	/* also to leave open the possibility of merging the two. */
	"BUTTON1DOWN":  "1024",
	"BUTTON2DOWN":  "2048",
	"BUTTON12DOWN": "4096",
	"BUTTON1UP":    "8192",
	"BUTTON2UP":    "16384",
	"BUTTON12UP":   "32768",
	"BUTTON1DRAG":  "65536",
	"BUTTON2DRAG":  "131072",
	"BUTTON12DRAG": "262144",
	"MOVING":       "524288",

	/* values for setmouse() and sweep() */
	"NOTHING":   "0",
	"ARROW":     "1",
	"SWEEP":     "2",
	"CROSS":     "3",
	"LEFTRIGHT": "4",
	"UPDOWN":    "5",
	"ANYWHERE":  "6",
	"BUSY":      "7",
	"DRAG":      "8",
	"BRUSH":     "9",
	"INVOKE":    "10",
	"POINT":     "11",
	"CLOSEST":   "12",
	"DRAW":      "13",
	/* values for cut() */
	"NORMAL":      "0",
	"TRUNCATE":    "1",
	"INCLUSIVE":   "2",
	"CUT_TIME":    "3",
	"CUT_FLAGS":   "4",
	"CUT_TYPE":    "5",
	"CUT_CHANNEL": "6",
	"CUT_NOTTYPE": "7",
	/* values for menudo() */
	"MENU_NOCHOICE":  "-1",
	"MENU_BACKUP":    "-2",
	"MENU_UNDEFINED": "-3",
	"MENU_MOVE":      "-4",
	"MENU_DELETE":    "-5",
	/* values for draw() */
	"CLEAR": "0",
	"STORE": "1",
	"XOR":   "2",
	/* values for window() */
	"TEXT":   "1",
	"PHRASE": "2",
	/* values for style() */
	"NOBORDER":      "0",
	"BORDER":        "1",
	"BUTTON":        "2",
	"MENUBUTTON":    "3",
	"PRESSEDBUTTON": "4",
	/* values for kill() signals */
	"KILL": "1",
}

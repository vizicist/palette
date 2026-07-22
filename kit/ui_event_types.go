package kit

const (
	uiStatusSubject          = "palette.local.ui.status"
	uiStepperSubject         = "palette.local.ui.stepper"
	uiCursorSubject          = "palette.local.ui.cursor"
	uiOBSSubject             = "palette.local.ui.obsrecord"
	uiSnapshotRequestSubject = "palette.local.ui.snapshot.request"
)

type EngineStatusSnapshot struct {
	Uptime            float64           `json:"uptime"`
	AttractMode       bool              `json:"attractmode"`
	OBSRunning        bool              `json:"obsrunning"`
	NATSConnected     bool              `json:"natsconnected"`
	NATSLocalRunning  bool              `json:"natslocalrunning"`
	NATSLocalURL      string            `json:"natslocalurl"`
	NATSWebsocket     string            `json:"natswebsocket"`
	Hostname          string            `json:"hostname"`
	Presets           map[string]string `json:"presets"`
	Mode              string            `json:"mode"`
	GuideDefaultLevel string            `json:"guidefaultlevel"`
	AttractAllowGUI   bool              `json:"attractallowgui"`
	Patches           map[string]string `json:"patches,omitempty"`
}

type OBSRecordUISnapshot struct {
	OBSRecordState
	OBSRunning        bool                `json:"obsrunning"`
	YouTubeConfigured bool                `json:"youtubeconfigured"`
	Upload            *YouTubeUploadState `json:"upload,omitempty"`
}

type UISnapshot struct {
	Status  EngineStatusSnapshot `json:"status"`
	Stepper *stepperStatus       `json:"stepper,omitempty"`
	Cursor  map[string]int64     `json:"cursor,omitempty"`
	OBS     OBSRecordUISnapshot  `json:"obsrecord"`
}

type UIEvent string

const (
	UIEventStatusChanged         UIEvent = "status.changed"
	UIEventStepperChanged        UIEvent = "stepper.changed"
	UIEventCursorActivityChanged UIEvent = "cursor_activity.changed"
	UIEventOBSRecordChanged      UIEvent = "obs_record.changed"
)

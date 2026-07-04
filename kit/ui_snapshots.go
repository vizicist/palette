package kit

func uiSnapshot() UISnapshot {
	return UISnapshot{
		Status:  uiStatusSnapshot(),
		Stepper: stepperStatusSnapshot(),
		Cursor:  cursorActivitySnapshot(),
		OBS:     obsRecordStatusSnapshot(),
	}
}

func uiStatusSnapshot() EngineStatusSnapshot {
	snapshot := EngineStatusSnapshot{
		Uptime:              Uptime(),
		AttractMode:         false,
		OBSRunning:          ObsIsRunning(),
		NATSConnected:       NatsIsConnected(),
		NATSLocalRunning:    EmbeddedNATSRunning(),
		NATSLeafConfigured:  EmbeddedNATSLeafConfigured(),
		NATSLeafConnections: EmbeddedNATSLeafConnections(),
		NATSLocalURL:        EmbeddedNATSURL(),
		NATSWebsocket:       EmbeddedNATSWebsocketURL(),
		Hostname:            Hostname(),
		Presets:             currentPresetSelectionSnapshot(),
		Mode:                CurrentMode(),
		GuideDefaultLevel:   GetParamWithDefault("global.guidefaultlevel", "0"),
		AttractAllowGUI:     IsTrueValue(GetParamWithDefault("global.attractallowgui", "false")),
	}
	if theAttractManager != nil {
		snapshot.AttractMode = theAttractManager.AttractModeIsOn()
	}
	if theQuad != nil {
		snapshot.Patches = map[string]string{}
		for _, patch := range patchNames {
			if p := Patchs[patch]; p != nil {
				snapshot.Patches[patch] = p.Status()
			}
		}
	}
	return snapshot
}

func stepperStatusSnapshot() *stepperStatus {
	if theStepper == nil {
		return nil
	}
	status, err := theStepper.StatusSnapshot()
	if err != nil {
		LogIfError(err)
		return nil
	}
	return &status
}

func cursorActivitySnapshot() map[string]int64 {
	if theCursorManager == nil {
		return nil
	}
	return theCursorManager.ActivitySnapshot()
}

func obsRecordStatusSnapshot() OBSRecordUISnapshot {
	return OBSRecordUISnapshot{
		OBSRecordState: ObsRecordStatusSnapshot(),
		OBSRunning:     ObsIsRunning(),
	}
}

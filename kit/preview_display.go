package kit

import "sync"

// Shared with python/spout_receiver.py. The manual-reset event carries state
// across receiver restarts without polling the engine or moving video over HTTP.
const spoutPreviewEventName = `Local\Palette.SpoutPreviewActive`
const spoutPerformanceEventName = `Local\Palette.SpoutPerformanceActive`

var (
	previewDisplayMutex         sync.Mutex
	writeSpoutPreviewActive     = platformWriteSpoutPreviewActive
	writeSpoutPerformanceActive = platformWriteSpoutPerformanceActive
)

func updateSpoutPreviewDisplay() error {
	previewDisplayMutex.Lock()
	defer previewDisplayMutex.Unlock()
	for _, patch := range Patchs {
		if patch.preview.isEnabled() {
			return writeSpoutPreviewActive(true)
		}
	}
	return writeSpoutPreviewActive(false)
}

package kit

import "sync"

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

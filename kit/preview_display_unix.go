//go:build !windows

package kit

func platformWriteSpoutPreviewActive(active bool) error {
	// Spout receivers run only on Windows.
	return nil
}

func platformWriteSpoutPerformanceActive(active bool) error {
	return nil
}

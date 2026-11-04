//go:build windows

package kit

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sys/windows"
)

// Shared with python/spout_receiver.py. These manual-reset events carry state
// across receiver restarts without polling the engine or moving video over HTTP.
const (
	spoutPreviewEventName     = `Local\Palette.SpoutPreviewActive`
	spoutPerformanceEventName = `Local\Palette.SpoutPerformanceActive`
)

type spoutDisplayEvent struct {
	once   sync.Once
	handle windows.Handle
	err    error
}

var spoutPreviewEvent, spoutPerformanceEvent spoutDisplayEvent

func (event *spoutDisplayEvent) write(eventName string, active bool) error {
	event.once.Do(func() {
		name, err := windows.UTF16PtrFromString(eventName)
		if err != nil {
			event.err = err
			return
		}
		// Keep the handle for the engine's lifetime. Receivers may start before
		// or after the engine, so an already-existing event is expected.
		handle, err := windows.CreateEvent(nil, 1, 0, name)
		if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			event.err = err
			return
		}
		event.handle = handle
	})
	if event.err != nil {
		return fmt.Errorf("Spout display state: %w", event.err)
	}
	if active {
		return windows.SetEvent(event.handle)
	}
	return windows.ResetEvent(event.handle)
}

func platformWriteSpoutPreviewActive(active bool) error {
	return spoutPreviewEvent.write(spoutPreviewEventName, active)
}

func platformWriteSpoutPerformanceActive(active bool) error {
	return spoutPerformanceEvent.write(spoutPerformanceEventName, active)
}

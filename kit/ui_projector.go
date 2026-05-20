package kit

import (
	"sync"

	json "github.com/goccy/go-json"
	"github.com/nats-io/nats.go"
)

var (
	uiEventBus           = NewEventBus[UIEvent]()
	uiProjectorStartOnce sync.Once
)

func StartUIStateProjector() {
	uiProjectorStartOnce.Do(func() {
		_ = NatsSubscribe(uiSnapshotRequestSubject, uiSnapshotRequestHandler)
		uiEventBus.Subscribe(UIEventStatusChanged, func() {
			publishUIState(uiStatusSubject, uiStatusSnapshot())
		})
		uiEventBus.Subscribe(UIEventStepperChanged, func() {
			publishUIState(uiStepperSubject, stepperStatusSnapshot())
		})
		uiEventBus.Subscribe(UIEventCursorActivityChanged, func() {
			publishUIState(uiCursorSubject, cursorActivitySnapshot())
		})
		uiEventBus.Subscribe(UIEventOBSRecordChanged, func() {
			publishUIState(uiOBSSubject, obsRecordStatusSnapshot())
		})
	})
	NotifyStatusChanged()
	NotifyStepperChanged()
	NotifyCursorActivityChanged()
	NotifyOBSRecordChanged()
}

func NotifyStatusChanged() {
	uiEventBus.Publish(UIEventStatusChanged)
}

func NotifyStepperChanged() {
	uiEventBus.Publish(UIEventStepperChanged)
}

func NotifyCursorActivityChanged() {
	uiEventBus.Publish(UIEventCursorActivityChanged)
}

func NotifyOBSRecordChanged() {
	uiEventBus.Publish(UIEventOBSRecordChanged)
}

func publishUIState(subject string, value any) {
	if value == nil {
		return
	}
	if err := NatsPublishJSON(subject, value); err != nil {
		LogOfType("nats", "publishUIState", "subject", subject, "err", err)
	}
}

func uiSnapshotRequestHandler(msg *nats.Msg) {
	bytes, err := json.Marshal(uiSnapshot())
	if err != nil {
		LogIfError(err)
		return
	}
	if err := msg.Respond(bytes); err != nil {
		LogIfError(err)
	}
}

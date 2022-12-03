package agent

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("logger", &Agent_logger{})
}

type Agent_logger struct{}

func (r *Agent_logger) Start(ctx *engine.AgentContext) {
}

func (r *Agent_logger) OnMidiEvent(ctx *engine.AgentContext, me engine.MidiEvent) {
	engine.Info("Agent_logger.onMidiEvent", "me", me)
}

func (r *Agent_logger) OnCursorEvent(ctx *engine.AgentContext, ce engine.CursorEvent) {
	engine.Info("Agent_logger.onCursorEvent", "ce", ce)
}

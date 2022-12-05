package agent

import (
	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("logger", &Agent_logger{})
}

type Agent_logger struct {
	ctx *engine.AgentContext
}

func (agent *Agent_logger) Start(ctx *engine.AgentContext) {
	agent.ctx = ctx
}

func (agent *Agent_logger) OnMidiEvent(me engine.MidiEvent) {
	engine.Info("Agent_logger.onMidiEvent", "me", me)
}

func (agent *Agent_logger) OnCursorEvent(ce engine.CursorEvent) {
	engine.Info("Agent_logger.onCursorEvent", "ce", ce)
}

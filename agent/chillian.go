package agent

/*
import (
	"fmt"

	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("chillian", &Agent_chillian{})
}

type Agent_chillian struct {
	ctx *engine.EngineContext
}

func (agent *Agent_chillian) Start(ctx *engine.EngineContext) {
	engine.Info("Agent_chillian.Start")
	agent.ctx = ctx
}

func (agent *Agent_chillian) Api(api string, apiargs map[string]string) (string, error) {
	return "", fmt.Errorf("Api in Chillian needs work")
}

func (agent *Agent_chillian) OnMidiEvent(me engine.MidiEvent) {
	ctx := agent.ctx
	ctx.Log("Agent_chillian.onMidiEvent", "me", me)
	// just echo it back out
	// ctx.ScheduleBytesNow(me.Bytes)
}

func (agent *Agent_chillian) OnCursorEvent(ce engine.CursorEvent) {
	ctx := agent.ctx
	if ce.Ddu == "down" { // || ce.Ddu == "drag" {

		channel := uint8(0)
		pitch := uint8(ce.X * 126.0)
		velocity := uint8(ce.Z * 1280)
		duration := 2 * engine.QuarterNote
		// synth := "0103 Ambient_E-Guitar"
		nt := engine.NewNoteFull(channel, pitch, velocity, duration)

		ctx.Log("Agent_chillian.OnCursorEvent", "pitch", pitch, "vel", velocity, "dur", duration)

		pe := &engine.PhraseElement{
			AtClick: ctx.CurrentClick(),
			Value:   nt,
		}
		synth := "P_03_C_03"
		phr := engine.NewPhrase().InsertElement(pe)
		ctx.SchedulePhrase(phr, ctx.CurrentClick(), synth)
	}
}

*/
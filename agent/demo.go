package agent

/*
import (
	"fmt"
	"time"

	"github.com/vizicist/palette/engine"
)

func init() {
	RegisterAgent("demo", &Agent_demo{})
}

type Agent_demo struct {
	ctx *engine.EngineContext
}

/////////////////////////// external interface

func (agent *Agent_demo) Start(ctx *engine.EngineContext) {
	agent.ctx = ctx
}

func (agent *Agent_demo) OnMidiEvent(me engine.MidiEvent) {
}

func (agent *Agent_demo) OnCursorEvent(ce engine.CursorEvent) {
	engine.Info("NewAgent_demo in OnCursorEvent!")
	if ce.Ddu == "down" {
		go func() {
			pe := agent.cursorToPhraseElement(ce)
			switch v := pe.Value.(type) {
			case *engine.NoteOn:
				engine.SendToSynth(pe)
			case *engine.NoteOff:
				engine.SendToSynth(pe)
			case *engine.NoteFull:
				engine.SendToSynth(pe)
				time.Sleep(2 * time.Second)
				noff := engine.NewNoteOff(v.Channel, v.Pitch, v.Velocity)
				peoff := &engine.PhraseElement{Value: noff}
				engine.SendToSynth(peoff)
			}
		}()
	}
	// engine.Info("OnCursorEvent in agent_demo called!", "ce", ce)
}

/////////////////////////// internal things

func (agent *Agent_demo) cursorToPhraseElement(ce engine.CursorEvent) *engine.PhraseElement {
	pitch := int(ce.X * 126.0)
	_ = pitch
	s := "+b"
	pe, err := engine.PhraseElementFromString((s))
	if err != nil {
		engine.LogError(err)
		pe, _ = engine.PhraseElementFromString("+a")
	}
	return pe
}

*/

package responder

import (
	"time"

	"github.com/vizicist/palette/engine"
)

func init() {
	engine.AddResponder("demo", &Responder_demo{})
	AllResponders["demo"] = &Responder_demo{}
}

type Responder_demo struct {
}

func NewResponder_demo() *Responder_demo {
	p := &Responder_demo{}
	return p
}

/////////////////////////// external interface

func (r *Responder_demo) OnMidiEvent(ctx *engine.ResponderContext, me engine.MidiEvent) {
}

func (r *Responder_demo) OnCursorEvent(ctx *engine.ResponderContext, ce engine.CursorEvent) {
	engine.Info("NewResponder_demo in OnCursorEvent!")
	if ce.Ddu == "down" {
		go func() {
			pe := r.cursorToPhraseElement(ce)
			switch v := pe.Value.(type) {
			case *engine.NoteOn:
				engine.SendPhraseElementToSynth(pe)
			case *engine.NoteOff:
				engine.SendPhraseElementToSynth(pe)
			case *engine.NoteFull:
				engine.SendPhraseElementToSynth(pe)
				time.Sleep(2 * time.Second)
				noff := engine.NewNoteOff(v.Pitch, v.Velocity, v.Synth)
				peoff := &engine.PhraseElement{Value: noff}
				engine.SendPhraseElementToSynth(peoff)
			}
		}()
	}
	// engine.Info("OnCursorEvent in responder_demo called!", "ce", ce)
}

/////////////////////////// internal things

func (r *Responder_demo) cursorToPhraseElement(ce engine.CursorEvent) *engine.PhraseElement {
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

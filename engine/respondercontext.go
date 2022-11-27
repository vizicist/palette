package engine

import (
	"fmt"
)

type ResponderContext struct {
	scheduler *Scheduler
	player    *Player
	// state     interface{} // for future responder-specific state
}

func NewResponderContext(player *Player) *ResponderContext {
	return &ResponderContext{
		scheduler: TheEngine().Scheduler,
		player:    player,
	}
}

func (ctx *ResponderContext) Log(msg string, keysAndValues ...interface{}) {
	Info(msg, keysAndValues...)
}

func (ctx *ResponderContext) MidiEventToPhrase(me MidiEvent) *Phrase {
	phr, err := ctx.player.MidiEventToPhrase(me)
	if err != nil {
		LogError(err)
		return nil
	} else {
		return phr
	}
}

func (ctx *ResponderContext) CurrentClick() Clicks {
	return CurrentClick()
}

func (ctx *ResponderContext) ScheduleDebug() string {
	return fmt.Sprintf("%s", ctx.scheduler)
}

func (ctx *ResponderContext) ScheduleNoteNow(pitch uint8, velocity uint8, duration Clicks, synth string) {
	Info("ctx.ScheduleNoteNow", "pitch", pitch)
	pe := &PhraseElement{Value: NewNoteFull(pitch, velocity, duration, synth)}
	phr := NewPhrase().InsertElement(pe)
	ctx.SchedulePhraseAt(phr, CurrentClick())
}

func (ctx *ResponderContext) SchedulePhraseNow(phr *Phrase) {
	ctx.SchedulePhraseAt(phr, CurrentClick())
}

func (ctx *ResponderContext) SchedulePhraseAt(phr *Phrase, click Clicks) {
	if phr == nil {
		Warn("ResponderContext.SchedulePhraseAt: phr == nil?")
		return
	}
	Info("ctx.SchedulePhraseAt", "click", click)
	go func() {
		se := &SchedElement{
			AtClick: click,
			Value:   phr,
		}
		ctx.scheduler.cmdInput <- ScheduleElementCmd{se}
	}()
}

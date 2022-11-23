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

func (ctx *ResponderContext) MidiEventToPhraseElement(me MidiEvent) *PhraseElement {
	nt, err := ctx.player.MidiEventToPhraseElement(me)
	if err != nil {
		LogError(err)
		return nil
	} else {
		return nt
	}
}

func (ctx *ResponderContext) CurrentClick() Clicks {
	return CurrentClick()
}

func (ctx *ResponderContext) ScheduleDebug() string {
	return fmt.Sprintf("%s", ctx.scheduler)
}

func (ctx *ResponderContext) SchedulePhraseNow(phr *Phrase) {
	ctx.SchedulePhraseAt(phr, CurrentClick())
}

func (ctx *ResponderContext) SchedulePhraseElementNow(pe *PhraseElement) {
	ctx.SchedulePhraseElementAt(pe, CurrentClick())
}

func (ctx *ResponderContext) SchedulePhraseAt(phr *Phrase, click Clicks) {
	if phr == nil {
		Warn("ResponderContext.SchedulePhraseAt: phr == nil?")
		return
	}
	go func() {
		ctx.scheduler.cmdInput <- SchedulePhraseCmd{phr, click}
	}()
}

func (ctx *ResponderContext) SchedulePhraseElementAt(pe *PhraseElement, click Clicks) {
	if pe == nil {
		Warn("ResponderContext.ScheduleNoteAt: nt == nil?")
		return
	}
	go func() {
		ctx.scheduler.cmdInput <- SchedulePhraseElementCmd{pe, click}
	}()
}

func (ctx *ResponderContext) ScheduleBytesAt(bytes []byte, click Clicks) {
	go func() {
		ctx.scheduler.cmdInput <- ScheduleBytesCmd{bytes, click}
	}()
}

func (ctx *ResponderContext) ScheduleBytesNow(bytes []byte) {
	go func() {
		ctx.scheduler.cmdInput <- ScheduleBytesCmd{bytes, ctx.CurrentClick()}
	}()
}

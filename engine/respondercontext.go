package engine

import (
	"fmt"
)

type ResponderContext struct {
	scheduler *Scheduler
	// state     interface{} // for future responder-specific state
}

func NewResponderContext() *ResponderContext {
	return &ResponderContext{
		scheduler: TheEngine().Scheduler,
	}
}

func (ctx *ResponderContext) Log(msg string, keysAndValues ...interface{}) {
	Info(msg, keysAndValues...)
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

func (ctx *ResponderContext) ScheduleNoteNow(nt *Note) {
	ctx.ScheduleNoteAt(nt, CurrentClick())
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

func (ctx *ResponderContext) ScheduleNoteAt(nt *Note, click Clicks) {
	if nt == nil {
		Warn("ResponderContext.ScheduleNoteAt: nt == nil?")
		return
	}
	phr := NewPhrase().InsertNote(nt)
	go func() {
		ctx.scheduler.cmdInput <- SchedulePhraseCmd{phr, click}
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

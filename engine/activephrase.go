package engine

import (
	"container/list"
	"fmt"
)

// ActivePhrase is a currently active MIDI phrase
type ActivePhrase struct {
	AtClick         Clicks
	SofarClick      Clicks
	phrase          *Phrase
	nextElement     *list.Element
	pendingNoteOffs *Phrase
}

// CallbackID xxx
type CallbackID int

func (ap *ActivePhrase) AdvanceByOneClick() (isDone bool) {

	thisClick := ap.SofarClick

	// See if any notes are due to be put out.

	// NOTE: we only send stuff if nextElement matches thisClick exactly
	for ; ap.nextElement != nil; ap.nextElement = ap.nextElement.Next() {

		pe := ap.nextElement.Value.(*PhraseElement)
		if pe.AtClick != thisClick {
			break
		}

		switch v := ap.nextElement.Value.(type) {
		case *NoteOn:
			Warn("advanceByOneClick can't handle NOTEON notes yet")
		case *NoteOff:
			Warn("advanceByOneClick can't handle NOTEOFF notes yet")
		case *NoteFull:
			newElement := pe.Copy()
			SendPhraseElementToSynth(newElement)

			newElement.AtClick = pe.AtClick + v.Duration
			ap.pendingNoteOffs.InsertElement(newElement)

		default:
			tstr := fmt.Sprintf("%T", v)
			Warn("advanceByOneClick unable to handle", "type", tstr)
		}
	}

	// Send whatever NOTEOFFs are due to be sent, and if everything has
	// been processed, delete it from the activePhrase
	isDone = ap.sendPendingNoteOffs(thisClick)
	ap.SofarClick++
	return isDone
}

// sendNoteOffs returns true if all of the pending notes and notesoff have been processed,
// i.e. the ActivePhrase can be removed
func (a *ActivePhrase) sendPendingNoteOffs(dueBy Clicks) (isDone bool) {

	if a.phrase == nil {
		Warn("ActivePhrase.sendPendingNoteOffs got unexpected nil phrase value")
		return true // pretend we're all done, so the broken ActivePhrase will get removed
	}

	// See if any of the Notes currently down are due, ie. occur before a.clickSoFar
	e := a.pendingNoteOffs.list.Front()
	for ; e != nil; e = e.Next() {

		pe := e.Value.(*PhraseElement)

		// It's ">= dueBy" so that 0-duration NoteFulls don't send NoteOff on the same click as the NoteOn
		if pe.EndOf() >= dueBy {
			break
		}

		SendPhraseElementToSynth(pe)

		a.pendingNoteOffs.list.Remove(e)
	}
	// Return true if there's nothing left to be processed in this ActivePhrase
	return (a.phrase.list.Front() == nil && a.pendingNoteOffs.list.Front() == nil)
}

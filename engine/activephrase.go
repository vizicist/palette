package engine

// ActivePhrase is a currently active MIDI phrase
type ActivePhrase struct {
	phrase          *Phrase
	startClick      Clicks
	clickSoFar      Clicks
	nextnote        *Note
	pendingNoteOffs *Phrase
	sid             string
}

// CallbackID xxx
type CallbackID int

// NoteOutputCallbackFunc xxx
type NoteOutputCallbackFunc func(n *Note)

// NewActivePhrase constructs a new ActivePhrase for a Phrase
func NewActivePhrase(p *Phrase, click Clicks, sid string) *ActivePhrase {
	return &ActivePhrase{
		phrase:          p,
		startClick:      click,
		sid:             sid,
		pendingNoteOffs: NewPhrase(),
	}
}

func (a *ActivePhrase) start() {
	if a.phrase == nil {
		Warn("ActivePhrase.start: Unexpected nil value for active.phrase")
	}
	a.clickSoFar = 0
	a.nextnote = a.phrase.firstnote // could be nil
}

func (a *ActivePhrase) AdvanceByOneClick() (isDone bool) {

	thisClick := a.clickSoFar
	Info("ActivePhrase being looked at", "phrase", a.phrase.ToString(), "started", a.startClick, "sofarclick", a.clickSoFar)
	// See if any notes in the Phrase are due to be put out.

	// NOTE: we only send stuff when thisClick is exactly the click it's scheduled at
	for n := a.nextnote; n != nil && n.Clicks == thisClick; n = n.next {
		switch n.TypeOf {
		case "noteon":
			Warn("ActivePhrasesManager.advanceActivePhrasesByOneStep can't handle NOTEON notes yet")
		case "noteoff":
			Warn("ActivePhrasesManager.advanceActivePhrasesByOneStep can't handle NOTEOFF notes yet")
		case "note":

			nd := n.Copy()
			nd.TypeOf = "noteon"
			SendNoteToSynth(nd)

			nd.TypeOf = "noteoff"
			nd.Clicks = n.EndOf()
			a.pendingNoteOffs.InsertNote(nd)
			Info("pendingNoteOffs after insert", "phrase", a.pendingNoteOffs.ToString())

		default:
			Warn("advanceActivePhrase unable to handle", "typeof", n.TypeOf)
		}
		// advance to the next note in the ActivePhrase
		a.nextnote = n.next
	}

	// Send whatever NOTEOFFs are due to be sent, and if everything has
	// been processed, delete it from the activePhrases
	isDone = a.sendPendingNoteOffs(thisClick)
	a.clickSoFar++
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
	ntoff := a.pendingNoteOffs.firstnote
	// XXX - not sure why it's "< dueBy", though it might be to
	// ensure that 0-duration notes don't send the ntoff on the same click as the nton.
	for ; ntoff != nil && ntoff.EndOf() < dueBy; ntoff = ntoff.next {

		SendNoteToSynth(ntoff)

		// Remove it from the notesDown phrase
		a.pendingNoteOffs.firstnote = ntoff.next
	}
	// Return true if there's nothing left to be processed in this ActivePhrase
	return (a.nextnote == nil && a.pendingNoteOffs.firstnote == nil)
}

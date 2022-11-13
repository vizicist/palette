package engine

import (
	"sync"
)

// ActivePhrase is a currently active MIDI phrase
type ActivePhrase struct {
	phrase          *Phrase
	startClick      Clicks
	clickSoFar      Clicks
	nextnote        *Note
	pendingNoteOffs *Phrase
}

// CallbackID xxx
type CallbackID int

// NoteOutputCallback is a call
type NoteOutputCallback struct {
	id       CallbackID
	Callback func(n *Note)
}

// NoteOutputCallbackFunc xxx
type NoteOutputCallbackFunc func(n *Note)

// NewActivePhrase constructs a new ActivePhrase for a Phrase
func NewActivePhrase(p *Phrase) *ActivePhrase {
	return &ActivePhrase{
		phrase:          p,
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

// ActivePhrasesManager manages ActivePhrases
type ActivePhrasesManager struct {
	mutex           sync.RWMutex
	activePhrases   map[string]*ActivePhrase // map of cursor ids to ActivePhrases
	outputCallbacks []*NoteOutputCallback
}

// NewActivePhrasesManager xxx
func NewActivePhrasesManager() *ActivePhrasesManager {
	mgr := &ActivePhrasesManager{
		activePhrases:   make(map[string]*ActivePhrase),
		outputCallbacks: make([]*NoteOutputCallback, 0),
	}
	return mgr
}

// StartPhrase xxx
// NOTE: startPhrase assumes that the mgr.mutex is held for writing
func (mgr *ActivePhrasesManager) StartPhraseAt(click Clicks, phrase *Phrase, cid string) {
	DebugLogOfType("phrase", "StartPhrase", "cid", cid)
	activePhrase, ok := mgr.activePhrases[cid]
	if !ok {
		activePhrase = NewActivePhrase(phrase)
	} else {
		// If active.note is non-nil, then we haven't sent the NoteOff
		// for this this cid.
		if activePhrase.phrase != nil {
			// This happens a lot when we get drag events
			mgr.StopPhrase(cid, activePhrase)
			// Don't need to remove it from r.activePhrases, the code below replaces it
		}
		activePhrase.phrase = phrase
	}
	activePhrase.startClick = click
	activePhrase.nextnote = phrase.firstnote // might be nil
	mgr.activePhrases[cid] = activePhrase
	activePhrase.start()
}

// StopPhrase xxx
// NOTE: stopPhrase assumes that the mgr.mutex is held for writing
func (mgr *ActivePhrasesManager) StopPhrase(cid string, active *ActivePhrase) {
	DebugLogOfType("phrase", "StopPhrase", "cid", cid)
	// If not provided in the arguments, look it up.
	if active == nil {
		var ok bool
		active, ok = mgr.activePhrases[cid]
		if !ok {
			// This occurs when an up cursor event is received
			// after the ActivePhrase is already finished and cleaned up.
			return
		}
	}

	readyToDelete := active.sendPendingNoteOffs(MaxClicks)
	if readyToDelete {
		delete(mgr.activePhrases, cid)
	}
}

// UncallbackOnOutput xxx
func (mgr *ActivePhrasesManager) UncallbackOnOutput(id CallbackID) {
	callbacks := make([]*NoteOutputCallback, 0)
	for _, cb := range mgr.outputCallbacks {
		if cb.id != id {
			callbacks = append(callbacks, cb)
		}
	}
	mgr.outputCallbacks = callbacks
}

// CallbackID xxx
var nextOutputCallbackID CallbackID = 1

// CallbackOnOutput xxx
func (mgr *ActivePhrasesManager) CallbackOnOutput(callback NoteOutputCallbackFunc) CallbackID {
	cb := &NoteOutputCallback{
		Callback: callback,
		id:       nextOutputCallbackID,
	}
	nextOutputCallbackID++
	mgr.outputCallbacks = append(mgr.outputCallbacks, cb)
	return cb.id
}

// AdvanceByOneClick xxx
func (mgr *ActivePhrasesManager) AdvanceByOneClick() {

	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	for cid, activePhrase := range mgr.activePhrases {
		if activePhrase.phrase == nil {
			Warn("advanceactivePhrases, unexpected phrase is nil", "cid", cid)
			// if activePhrase.sendPendingNoteOffs(MaxClicks) {
			// 	delete(mgr.activePhrases, cid)
			// }
		} else {
			isDone := activePhrase.AdvanceByOneClick()
			if isDone {
				delete(mgr.activePhrases, cid)
			}
		}
	}
}

func (mgr *ActivePhrasesManager) terminateActiveNotes() {
	mgr.mutex.RLock()
	defer mgr.mutex.RUnlock()
	for id, a := range mgr.activePhrases {
		if a != nil {
			a.sendPendingNoteOffs(a.clickSoFar)
		} else {
			Warn("Hey, nil activeNotes entry", "id", id)
		}
	}
}

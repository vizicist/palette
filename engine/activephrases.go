package engine

import (
	"log"
	"sync"
)

// ActivePhrase is a currently active MIDI phrase
type ActivePhrase struct {
	phrase          *Phrase
	clickSoFar      Clicks
	nextnote        *Note
	pendingNoteOffs *Phrase
}

// ActivePhrasesManager manages ActivePhrases
type ActivePhrasesManager struct {
	ActivePhrasesMutex sync.RWMutex
	activePhrases      map[string]*ActivePhrase // map of cursor ids to ActivePhrases
	outputCallbacks    []*NoteOutputCallback
	midiOutput         MidiDevice
}

// NewActivePhrase constructs a new ActivePhrase for a Phrase
func NewActivePhrase(p *Phrase) *ActivePhrase {
	return &ActivePhrase{
		phrase:          p,
		pendingNoteOffs: NewPhrase(),
	}
}

// NewActivePhrasesManager xxx
func NewActivePhrasesManager(midiOutput MidiDevice) *ActivePhrasesManager {
	mgr := &ActivePhrasesManager{
		activePhrases:   make(map[string]*ActivePhrase),
		outputCallbacks: make([]*NoteOutputCallback, 0),
		midiOutput:      midiOutput,
	}
	return mgr
}

func (a *ActivePhrase) start() {
	if a.phrase == nil {
		log.Printf("ActivePhrase.start: Unexpected nil value for active.phrase\n")
	}
	a.clickSoFar = 0
	a.nextnote = a.phrase.firstnote // could be nil
}

// sendNoteOffs returns true if all of the pending notes and notesoff have been processed,
// i.e. the ActivePhrase can be removed
func (a *ActivePhrase) sendNoteOffs(mididevice MidiDevice, due Clicks, debug bool, callbacks []*NoteOutputCallback) bool {

	if a.phrase == nil {
		log.Printf("ActivePhrase.sendNoteOffs got unexpected nil phrase value\n")
		return true // pretend we're all done, so the broken ActivePhrase will get removed
	}

	// See if any of the Notes currently down are due, ie. occur before a.clickSoFar
	ntoff := a.pendingNoteOffs.firstnote
	for ; ntoff != nil && ntoff.EndOf() < due; ntoff = ntoff.next {
		// log.Printf("ntdown=%s is sending its NOTEOFF!\n", ntdown)
		mididevice.SendNote(ntoff, debug, callbacks)
		// Remove it from the notesDown phrase
		a.pendingNoteOffs.firstnote = ntoff.next
	}
	// Return true if there's nothing left to be processed in this ActivePhrase
	return (a.nextnote == nil && a.pendingNoteOffs.firstnote == nil)
}

// StartPhrase xxx
// NOTE: startPhrase assumes that the r.activePhrasesMutex is held for writing
func (mgr *ActivePhrasesManager) StartPhrase(p *Phrase, cid string) {
	active, ok := mgr.activePhrases[cid]
	if !ok {
		active = NewActivePhrase(p)
	} else {
		// If active.note is non-nil, then we haven't sent the NoteOff
		// for this this cid.
		if active.phrase != nil {
			// This happens a lot when we get drag events
			mgr.StopPhrase(cid, active, false)
			// Don't need to remove it from r.activePhrases, the code below replaces it
		}
		active.phrase = p
	}
	active.nextnote = p.firstnote // might be nil
	mgr.activePhrases[cid] = active
	active.start()
}

// StopPhrase xxx
// NOTE: stopPhrase assumes that the r.activePhrasesMutex is held for writing
func (mgr *ActivePhrasesManager) StopPhrase(cid string, active *ActivePhrase, forceDelete bool) {

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

	readyToDelete := active.sendNoteOffs(mgr.midiOutput, MaxClicks, DebugUtil.MIDI, mgr.outputCallbacks)
	if readyToDelete || forceDelete {
		delete(mgr.activePhrases, cid)
	}
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

// AdvanceActivePhrasesByOneStep xxx
func (mgr *ActivePhrasesManager) AdvanceActivePhrasesByOneStep() {

	mgr.ActivePhrasesMutex.Lock()
	defer mgr.ActivePhrasesMutex.Unlock()

	for cid, a := range mgr.activePhrases {
		if a.phrase == nil {
			log.Printf("advanceactivePhrases, unexpected phrase is nil for cid=%s?  deleting it\n", cid)
			if a.sendNoteOffs(mgr.midiOutput, MaxClicks, DebugUtil.MIDI, mgr.outputCallbacks) {
				delete(mgr.activePhrases, cid)
			}
			continue
		}

		// log.Printf("ActivePhrase for cid=%s a=%p a.phrase=%s a.nextnote=%p\n", cid, a, a.phrase, a.nextnote)
		n := a.nextnote
		// See if any notes in the Phrase are due to be put out
		for ; n != nil && n.Clicks <= a.clickSoFar; n = n.next {
			switch n.TypeOf {
			case NOTEON:
				log.Printf("Reactor.advanceActivePhrasesByOneStep can't handle NOTEON notes yet\n")
			case NOTEOFF:
				log.Printf("Reactor.advanceActivePhrasesByOneStep can't handle NOTEOFF notes yet\n")
			case NOTE:
				nd := n.Copy()
				nd.TypeOf = NOTEON
				mgr.midiOutput.SendNote(nd, DebugUtil.MIDI, mgr.outputCallbacks)
				nd.TypeOf = NOTEOFF
				nd.Clicks = n.EndOf()
				a.pendingNoteOffs.InsertNote(nd)
			default:
				log.Printf("advanceActivePhrase unable to handle n.Typeof=%d n=%s\n", n.TypeOf, n)
			}
			// advance to the next note in the ActivePhrase
			a.nextnote = n.next
		}

		// Send whatever NOTEOFFs are due to be sent, and if everything has
		// been processed, delete it from the activePhrases
		if a.sendNoteOffs(mgr.midiOutput, a.clickSoFar, DebugUtil.MIDI, mgr.outputCallbacks) {
			delete(mgr.activePhrases, cid)
		}
		// log.Printf("Advancing clickSoFar to %d\n", a.clickSoFar)
		a.clickSoFar++
	}
}

package engine

// Copy returns a copy of a Phrase
func (p *Phrase) Copy() *Phrase {

	p.RLock()
	defer p.RUnlock()

	r := NewPhrase()
	for n := p.firstnote; n != nil; n = n.next {
		nn := n.Copy()
		r.InsertNote(nn)
	}
	r.Length = p.Length
	return r
}

// CopyAndAppend makes a copy of a Note and appends it to the Phrase
func (p *Phrase) CopyAndAppend(n *Note) *Note {
	newn := n.Copy()
	if p.firstnote == nil {
		p.firstnote = newn
	} else {
		p.lastnote.next = newn
	}
	p.lastnote = newn
	return newn
}

// CutTime creates a new Phrase with notes in a given time range
func (p *Phrase) CutTime(fromclick, toclick Clicks) *Phrase {

	p.RLock()
	defer p.RUnlock()

	newp := NewPhrase()
	for n := p.firstnote; n != nil; n = n.next {
		if n.Clicks >= fromclick && n.Clicks < toclick {
			newp.CopyAndAppend(n)
		}
	}
	newp.ResetLengthNoLock()

	return newp
}

// CutSound creates a new Phrase with notes for a given sound
func (p *Phrase) CutSound(sound string) *Phrase {

	p.RLock()
	defer p.RUnlock()

	newp := NewPhrase()
	for n := p.firstnote; n != nil; n = n.next {
		if n.Sound == sound {
			newp.CopyAndAppend(n)
		}
	}
	newp.ResetLengthNoLock()

	return newp
}

// AdjustTimes returns a new Phrase shifted by shift Clicks
func (p *Phrase) AdjustTimes(shift Clicks) *Phrase {

	p.RLock()
	defer p.RUnlock()

	ret := NewPhrase()
	for n := p.firstnote; n != nil; n = n.next {
		newn := ret.CopyAndAppend(n)
		newn.Clicks += shift
	}
	ret.ResetLengthNoLock()

	return ret
}

// Merge merges a second Phrase into a Phrase
// NOTE: we get a Write lock on the Phrase,
// since we're actually changing it.
func (p *Phrase) Merge(fromPhrase *Phrase) *Phrase {

	p.Lock() // write lock, we're changing p
	defer p.Unlock()

	for nt := fromPhrase.firstnote; nt != nil; nt = nt.next {
		nn := nt.Copy()
		p.InsertNote(nn)
	}
	p.ResetLengthNoLock()
	return p
}

// Arpeggio returns an arpeggiated version of the phrase.
// One way of describing is that all the notes have been
// separated and then put back together, back-to-back.
func (p *Phrase) Arpeggio() *Phrase {

	p.RLock()
	defer p.RUnlock()

	lastend := Clicks(0)
	r := NewPhrase()
	for nt := p.firstnote; nt != nil; nt = nt.next {
		nn := nt.Copy()
		nn.Clicks = lastend
		r.InsertNote(nn)
		d := nt.Duration
		if d == 0 {
			d = 1
		}
		lastend += d
	}
	r.Length = lastend
	return r
}

// Step returns a stepped version of the Phrase.
func (p *Phrase) Step(stepsize Clicks) *Phrase {

	p.RLock()
	defer p.RUnlock()

	first := true
	lasttime := Clicks(0)
	steptime := Clicks(0)
	r := NewPhrase()
	for nt := p.firstnote; nt != nil; nt = nt.next {
		// Notes that are at the same time (like chords)
		// are still at the same time.
		if !first && nt.Clicks != lasttime {
			steptime += stepsize
			lasttime = nt.Clicks
		}
		first = false
		newnt := nt.Copy()
		newnt.Clicks = steptime
		newnt.Duration = stepsize
		r.InsertNote(newnt)
	}
	r.Length = steptime + stepsize
	return (r)
}

// Transpose returns a Phrase whose pitch is transposed.
func (p *Phrase) Transpose(delta int) *Phrase {

	p.RLock()
	defer p.RUnlock()

	r := NewPhrase()
	for nt := p.firstnote; nt != nil; nt = nt.next {
		newnt := r.CopyAndAppend(nt)
		newnt.Pitch = uint8(int(newnt.Pitch) + delta)
	}
	return r
}

// LowestPitch returns the lowest pitch in a Phrase
func (p *Phrase) LowestPitch(delta int) uint8 {

	p.RLock()
	defer p.RUnlock()

	lowest := uint8(127)
	for nt := p.firstnote; nt != nil; nt = nt.next {
		if nt.Pitch < lowest {
			lowest = nt.Pitch
		}
	}
	return lowest
}

// Legato extends the duration of each note to abutt the start of the next note.
// Doesn't modify the duration of the last note.
func (p *Phrase) Legato() *Phrase {
	r := p.Copy()
	for nt := r.firstnote; nt != nil; nt = nt.next {
		if nt.IsNote() {
			nextt := r.NextTime(nt.Clicks)
			// notes at the end of the phrase aren't touched
			if nextt >= 0 {
				nt.Duration = nextt - nt.Clicks
			}
		}
	}
	return r
}

// AtTime returns those notes in the specified phrase that are
//sounding at the specified time.  If a note ends exactly
//at the specified time, it is not included.
func (p *Phrase) AtTime(tm Clicks) *Phrase {

	p.RLock()
	defer p.RUnlock()

	newp := NewPhrase()
	for n := p.firstnote; n != nil; n = n.next {
		if n.Clicks <= tm && n.EndOf() > tm {
			// Assumes Phrase is already sorted, so always append to end of new phrase
			newp.Append(n.Copy())
		}
	}
	newp.ResetLengthNoLock()
	return newp
}

// NextTime returns the time of the next note AFTER time st.
// If there are no notes after it, returns -1.
func (p *Phrase) NextTime(st Clicks) Clicks {
	p.RLock()
	defer p.RUnlock()

	nexttime := Clicks(-1)
	for nt := p.firstnote; nt != nil; nt = nt.next {
		if nt.Clicks > st {
			nexttime = nt.Clicks
			break
		}
	}
	return nexttime
}

/*
// Scadjust returns a Phrase where notes have been adjusted
// to be on a particular Scale
func (p *Phhrase) Scadjust(mel,scale) {
	r := NewPhrase()
	scarr = []
	for ( nt in scale )
		scarr[canonic(nt)] = 1
	for ( nt in mel ) {
		if ( nt.type & (NOTE|NOTEOFF|NOTEON) ) {
			inc = sign = 1
			cnt = 0
			# Don't do computation with nt.pitch directly,
			# because negative pitches are invalid
			# and get adjusted automatically
			ptch = nt.pitch
			while ( ! (canonic(ptch) in scarr) && cnt++ < 100 ) {
				ptch += (sign*inc)
				inc = inc + 1
				sign = -sign
			}
			nt.pitch = ptch
			if ( cnt >= 100 ) {
				print("Something's amiss in scadjust, for nt=",nt)
				continue
			}
		}
		r |= nt
	}
	return(r)
}
*/

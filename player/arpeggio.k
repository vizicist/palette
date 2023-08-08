#name	arpeggio
#usage	arpeggio(phrase)
#desc	Returns an arpeggiated version of the phrase.  One way of describing
#desc	it is that all the notes have been separated and then put back
#desc	together, back-to-back.

function arpeggio(phr) {
	if ( nargs() < 1 ) {
		print("usage: arpeggio(phrase)")
		return('')
	}
	lastend = 0
	r = ''
	for ( nt in phr ) {
		nt.time = lastend
		r |= nt
		d = nt.duration
		if ( d == 0 )
			d = 1
		lastend += d
	}
	r.length = lastend
	return(r)
}

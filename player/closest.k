#name	closest
#usage	closest(note,scale [,direction] )
#desc	Returns a note from the specified scale that is closest in pitch
#desc	to the specified note.  If the optional direction argument is given,
#desc	it specifies the direction (1==up, -1==down) that we want the
#desc	chosen note to be in (relative to the original note).  Specifying
#desc	the direction also guarantees that a note different from the
#desc	original is chosen (if possible).

function closest(nt,scl,dir) {
	if ( nargs() < 3 )
		dir = 0
	if ( sizeof(scl) == 0 ) {
		print("Hey, closest() needs some notes in the scale!")
		return('')
	}
	inc = sign = 1
	arr = []
	for ( n in scl )
		arr[n.pitch % 12] = 1
	t = nt.pitch
	if ( dir != 0 ) {
		t += dir
		inc++
		sign = -dir
	}
	while  ( ( ! (t%12) in arr) || (dir!=0 && dir==sign) || t<0 ) {
		t += (sign*inc++)
		# make sure, if we go past the ends, that we aren't
		# looking in that direction, cause we'd never find anything
		if ( t < 0 )
			dir = 1
		if ( t > 127 )
			dir = -1
		sign = -sign
	}
	nt.pitch = t
	return(nt)
}

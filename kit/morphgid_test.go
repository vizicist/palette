package kit

import "testing"

func newTestMorph(t *testing.T) *oneMorph {
	t.Helper()
	old := theCursorManager
	t.Cleanup(func() { theCursorManager = old })
	theCursorManager = NewCursorManager()
	return &oneMorph{contacts: map[int]morphContact{}}
}

// A contact staying on one source keeps one GID for the whole gesture.
func TestGidForContactIsStableWithinASource(t *testing.T) {
	m := newTestMorph(t)

	first, retire, _ := m.gidForContact(0, "A")
	if first == 0 {
		t.Fatal("a GID of 0 is never issued")
	}
	if retire != 0 {
		t.Fatalf("a brand new contact retired GID %d", retire)
	}
	for i := 0; i < 5; i++ {
		gid, retire, _ := m.gidForContact(0, "A")
		if gid != first {
			t.Fatalf("frame %d: GID changed from %d to %d without changing source", i, first, gid)
		}
		if retire != 0 {
			t.Fatalf("frame %d: retired GID %d without changing source", i, retire)
		}
	}
}

// Crossing to another source starts a new gesture, retires the old one, and -
// the actual regression - the new GID sticks. It used to be minted and thrown
// away, so the next frame reverted to the old GID and the contact alternated
// between two of them for the rest of the drag.
func TestGidForContactStoresTheGidMintedOnACrossing(t *testing.T) {
	m := newTestMorph(t)

	before, _, _ := m.gidForContact(0, "A")

	crossed, retireGid, retireTag := m.gidForContact(0, "B")
	if crossed == before {
		t.Fatal("crossing to another source did not start a new gesture")
	}
	if retireGid != before {
		t.Fatalf("crossing retired GID %d, want the old gesture's %d", retireGid, before)
	}
	if retireTag != "A" {
		t.Fatalf("crossing retired tag %q, want %q", retireTag, "A")
	}

	// The frame after the crossing must stay on the new GID.
	after, retire, _ := m.gidForContact(0, "B")
	if after != crossed {
		t.Fatalf("after the crossing the GID reverted from %d to %d", crossed, after)
	}
	if retire != 0 {
		t.Fatalf("a second frame on the same source retired GID %d", retire)
	}
}

// Two contacts on different sources must not disturb each other. The source was
// tracked in one previousTag field on the Morph, and currentTag is recomputed
// per contact, so on a quadrants Morph two fingers in different quadrants made
// every single event look like a crossing.
func TestGidForContactTracksSourcesPerContact(t *testing.T) {
	m := newTestMorph(t)

	gidA, _, _ := m.gidForContact(0, "A")
	gidB, _, _ := m.gidForContact(1, "B")
	if gidA == gidB {
		t.Fatal("two contacts were given the same GID")
	}

	for i := 0; i < 5; i++ {
		got, retire, _ := m.gidForContact(0, "A")
		if got != gidA || retire != 0 {
			t.Fatalf("frame %d: contact 0 got GID %d (retire %d), want %d and no retire",
				i, got, retire, gidA)
		}
		got, retire, _ = m.gidForContact(1, "B")
		if got != gidB || retire != 0 {
			t.Fatalf("frame %d: contact 1 got GID %d (retire %d), want %d and no retire",
				i, got, retire, gidB)
		}
	}
}

// A reused contact slot after an up starts a fresh gesture rather than
// resurrecting the finished one.
func TestGidForContactAfterSlotIsReleased(t *testing.T) {
	m := newTestMorph(t)

	first, _, _ := m.gidForContact(0, "A")
	delete(m.contacts, 0) // what readFrames does on ddu == "up"

	second, retire, _ := m.gidForContact(0, "A")
	if second == first {
		t.Fatal("a reused contact slot resurrected the previous gesture's GID")
	}
	if retire != 0 {
		t.Fatalf("a reused slot retired GID %d, but its gesture already ended", retire)
	}
}

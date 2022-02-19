#include "NosuchException.h"
#include "NosuchUtil.h"
#include "SharedMemHeader.h"
#include "blob.h"
#include "BlobContour.h"

buff_index
MMTT_SharedMemHeader::grab_unused_buffer()
{
	buff_index found = BUFF_UNSET;
	for ( int b=0; b<NUM_BUFFS; b++ ) {
		if ( ! buff_inuse[b] ) {
			found = b;
			break;
		}
	}
	if ( found != BUFF_UNSET ) {
		buff_inuse[found] = true;
	}
	return found;
}

void
MMTT_SharedMemHeader::init() 
{
    // Magic number to make sure we are looking at the correct memory
    // must be set to TOP_SHM_MAGIC_NUMBER (0xe95df673)
    magicNumber = MMTT_SHM_MAGIC_NUMBER; 
    // header, must be set to TOP_SHM_VERSION_NUMBER
    version = MMTT_SHM_VERSION_NUMBER;

	for ( buff_index b=0; b<NUM_BUFFS; b++ ) {
		// cursorsOffset[b] = calcCursorOffset(b);
		outlinesOffset[b] = calcOutlineOffset(b);
		pointsOffset[b] = calcPointOffset(b);
		buff_inuse[b] = false;
		clear_lists(b);
	}

	ncursors_max = MMTT_CURSORS_MAX;
	noutlines_max = MMTT_OUTLINES_MAX;
	npoints_max = MMTT_POINTS_MAX;

	// These values need to be locked, when looked at or changed
	buff_being_constructed = BUFF_UNSET;
	buff_displayed_last_time = BUFF_UNSET;
	buff_to_display_next = BUFF_UNSET;

	seqnum = 0;

}

int
MMTT_SharedMemHeader::addCursorOutline(buff_index b,
	int rid, int sid, float x, float y, float z, int npoints) {

	NosuchAssert(b!=BUFF_UNSET);

	int cnum = numoutlines[b];
	// CursorSharedMem* cm = cursor(b,cnum);
	OutlineMem* om = outline(b,cnum);

	// NOTE: region id's in the shared memory are shifted down by 1
	om->region = rid-1;
	om->sid = sid;
	om->x = x;
	om->y = y;
	om->z = z;
	om->npoints = npoints;
	om->index_of_firstpoint = numpoints[b];
	// NosuchDebug("SHARED MEM - Adding Outline npoints=%d",om->npoints);

	numoutlines[b]++;

	return cnum;
}

int
MMTT_SharedMemHeader::addPoint(buff_index b, float x, float y, float z) {
	NosuchAssert(b!=BUFF_UNSET);
	int pnum = numpoints[b];
	PointMem* p = point(b,pnum);
	p->x = x;
	p->y = y;
	p->z = z;
	numpoints[b]++;
	// NosuchDebug("SHARED MEM - Adding Point xy=%.4f,%.4f",p->x,p->y);
	// NosuchDebug("MMTT_Kinect - Added point buff=%d pnum=%d xy=%f,%f",b,pnum,x,y);
	return pnum;
}

#if 0
void
print_buff_info(char *prefix, MMTT_SharedMemHeader* h)
{
	NosuchDebug(">>>>> %s BUFF INFO",prefix);
	for ( int b=0; b<3; b++ ) {
		NosuchDebug(">>>>>   buff=%d numoutlines=%d numpoints=%d",
			b,h->numoutlines[b],h->numpoints[b]);
		for ( int i=0; i<h->numoutlines[b]; i++ ) {
			OutlineMem* om = h->outline(b,i);
			NosuchDebug(">>>>>       outline=%d region=%d numpoints=%d",
				i,om->region,om->npoints);
			if ( om->npoints > 100000 ) {
				NosuchDebug("CORRUPTION in Buffer!! b=%d oi=%d  om=%lx",
					b,i,(long)om);
			}
		}
	}
	NosuchDebug(">>>>> END OF %s BUFF INFO",prefix);
}
#endif

void
MMTT_SharedMemHeader::clear_lists(buff_index buffnum) {
	NosuchAssert(buffnum!=BUFF_UNSET);
	numoutlines[buffnum] = 0;
	numpoints[buffnum] = 0;
}

void
MMTT_SharedMemHeader::check_sanity() {
	NosuchDebug("MMTT_SharedMemHeader::check_sanity()");
	int nused = 0;
	if ( buff_inuse[0] )
		nused++;
	if ( buff_inuse[1] )
		nused++;
	if ( buff_inuse[2] )
		nused++;

	bool ptr_used[3];
	ptr_used[0] = false;
	ptr_used[1] = false;
	ptr_used[2] = false;
	if ( buff_being_constructed != BUFF_UNSET ) {
		ptr_used[buff_being_constructed] = true;
	}
	if ( buff_displayed_last_time != BUFF_UNSET ) {
		ptr_used[buff_displayed_last_time] = true;
	}
	if ( buff_to_display_next != BUFF_UNSET ) {
		ptr_used[buff_to_display_next] = true;
	}

	int nptrused = 0;
	if ( ptr_used[0] )
		nptrused++;
	if ( ptr_used[1] )
		nptrused++;
	if ( ptr_used[2] )
		nptrused++;

	NosuchAssert(nused != nptrused);
}

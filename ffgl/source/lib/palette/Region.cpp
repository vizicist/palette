#include  <stdlib.h>
#include <vector>
#include <string>
#include <cstdlib> // for srand, rand

#include "PaletteAll.h"

#define DEFINE_TYPES(t) std::vector<std::string> RegionParams_##t##Types;
#include "RegionParams_types.h"

Region::Region() {

	_spritelist = new SpriteList();

	NosuchLockInit(&_region_mutex,"region");
	_cursorlist_rwlock = PTHREAD_RWLOCK_INITIALIZER;
	int rc = pthread_rwlock_init(&_cursorlist_rwlock, NULL);
	if ( rc ) {
		NosuchDebug("Failure on pthread_rwlock_init!? rc=%d",rc);
	}

	x_min = 0.00f;
	y_min = 0.00f;
	x_max = 1.0f;
	y_max = 1.0f;

	initParams();

}

Region::~Region() {
	NosuchDebug(1,"Region DESTRUCTOR!");
}

void Region::initParams() {
	numalive = 0;
	debugcount = 0;
	last_tm = 0;
	leftover_tm = 0;
	// fire_period = 10;  // milliseconds
	fire_period = 1;  // milliseconds
	onoff = 0;
}

TrackedCursor* Region::_getTrackedCursor(std::string cid, std::string cidsource) {
	TrackedCursor* retc = NULL;

	for ( std::list<TrackedCursor*>::iterator i = _cursors.begin(); i!=_cursors.end(); i++ ) {
		TrackedCursor* c = *i;
		NosuchAssert(c);
		if (c->cid() == cid && c->cidsource() == cidsource) {
			retc = c;
			break;
		}
	}
	return retc;
}

float Region::_maxCursorDepth() {
	// We assume the cursorlist is locked
	float maxval = 0;
	for ( std::list<TrackedCursor*>::iterator i = _cursors.begin(); i!=_cursors.end(); i++ ) {
		TrackedCursor* c = *i;
		NosuchAssert(c);
		float d = c->curr_raw_depth;
		if ( d > maxval )
			maxval = d;
	}
	return maxval;
}

void
Region::setTrackedCursor(Palette* palette, std::string cid, std::string cidsource, glm::vec2 pos, float z) {

	if ( pos.x < x_min || pos.x > x_max || pos.y < y_min || pos.y > y_max ) {
		NosuchDebug("Ignoring out-of-bounds cursor pos=%f,%f,%f\n",pos.x,pos.y);
		return;
	}

	if ( ! cursorlist_lock_write() ) {
		NosuchDebug("Region::setTrackedCursor, unable to lock cursorlist");
		return;
	}

	TrackedCursor* c = _getTrackedCursor(cid,cidsource);
	if ( c != NULL ) {
		c->settargetpos(pos);
		c->set_target_depth(z);
		// c->setarea(z);
	} else {
		c = new TrackedCursor(palette, cid, cidsource, this, pos, z);
		if (NosuchDebugCursor) {
			NosuchDebug("Region.setTrackedCursor: new TrackedCursor cid=%s", cid.c_str());
		}
		_cursors.push_back(c);
	}
	c->touch();

	cursorlist_unlock();

	return;
}

float Region::getMoveDir(std::string movedirtype) {
	if ( movedirtype == "left" ) {
		return 180.0f;
	}
	if ( movedirtype == "right" ) {
		return 0.0f;
	}
	if ( movedirtype == "up" ) {
		return 270.0f;
	}
	if ( movedirtype == "down" ) {
		return 90.0f;
	}
	if ( movedirtype == "random" || movedirtype == "cursor" ) {
		return 360.0f * RANDFLOAT;
	}
	if ( movedirtype == "random90" ) {
		return 90.0f * (rand() % 4);
	}
	if ( movedirtype == "updown" ) {
		return 90.0f + 180.0f * (rand() % 2);
	}
	if ( movedirtype == "leftright" ) {
		return 180.0f * (rand() % 2);
	}
	// throw NosuchException("Unrecognized movedirtype value %s",movedirtype.c_str());
	throw NosuchBadValueException();
}

void Region::doCursorUp(Palette* palette, std::string cid) {

	if (!cursorlist_lock_write()) {
		NosuchDebug("Region::doCursorUp, unable to lock cursorlist");
		return;
	}
	bool found = false;
	if (NosuchDebugCursor) {
		NosuchDebug("Region.doCursorUp cid=%s\n", cid.c_str());
	}
	for (std::list<TrackedCursor*>::iterator i = _cursors.begin(); i != _cursors.end(); ) {
		TrackedCursor* c = *i;
		NosuchAssert(c);
		if (NosuchDebugCursor) {
			NosuchDebug("Region.doCursorUp TrackedCursor loop c->cid=%s\n", c->cid().c_str());
		}
		if (c->cid() == cid) {
			found = true;
			if (NosuchDebugCursor) {
				NosuchDebug("Region.doCursorUp: deleting cid=%s",cid.c_str());
			}
			i = _cursors.erase(i);
			delete c;
			break;
		}
		i++;
	}
	if (NosuchDebugCursor) {
		if (!found) {
			NosuchDebug("Region.doCursorUp: didn't find cursor cid=%s", cid.c_str());
		}
		NosuchDebug("End of doCursorUp, _cursors.size = %d", _cursors.size());
	}
	cursorlist_unlock();
}

void Region::clearCursors() {

	if (!cursorlist_lock_write()) {
		NosuchDebug("Region::clearCursors, unable to lock cursorlist");
		return;
	}
	// NosuchDebug("Begin clearCursors, _cursors.size = %d",_cursors.size());
	for (std::list<TrackedCursor*>::iterator i = _cursors.begin(); i != _cursors.end(); ) {
		TrackedCursor* c = *i;
		NosuchAssert(c);
		if (NosuchDebugCursor) {
			NosuchDebug("Region.clearCursor: deleting cid=%s",c->cid().c_str());
		}
		i = _cursors.erase(i);
		delete c;
	}
	// NosuchDebug("End clearCursors, _cursors.size = %d",_cursors.size());
	cursorlist_unlock();
}

float
Region::spriteMoveDir(TrackedCursor* c)
{
	float dir;
	if (params.movedir == "cursor") {
		if (c != NULL) {
			dir = c->curr_degrees;
		}
		else {
			dir = 360.0f * RANDFLOAT;
		}
		// NosuchDebug("Region::spriteMoveDir cursor! dir=%f", dir);
		// NosuchDebug("spriteMoveDir cursor degrees = %f",c->curr_degrees);
		// not sure why I have to reverse it - the cursor values are probably reversed
		dir -= 90.0;
		if (dir < 0.0) {
			dir += 360.0;
		}
	}
	else {
		dir = getMoveDir(params.movedir);
		// NosuchDebug("Region::spriteMoveDir movedir=%s dir=%f", params.movedir.c_str(), dir);
	}
	// NosuchDebug("spriteMoveDir dir=%f movedir=%s", dir, params.movedir.c_str());
	return dir;
}

Sprite*
Region::makeSprite(std::string shape) {

	Sprite* s = NULL;
	if (shape == "square") {
		s = new SpriteSquare();
		// NosuchDebug("NEW SpriteSquare initial size=%lf width=%lf depth=%lf\n", s->state.size,s->width(), s->state.depth);
	}
	else if (shape == "triangle") {
		s = new SpriteTriangle();
	}
	else if (shape == "circle") {
		s = new SpriteCircle();
	}
	else if (shape == "line") {
		s = new SpriteLine();
	}
	else if (shape == "nothing") {
		//
	}
	else {
		throw NosuchUnrecognizedTypeException();
	}
	return s;
}

void
Region::instantiateSprite(TrackedCursor* c, bool throttle) {

	std::string shape = params.shape;

	if (params.spritesource != "cursor") {
		// NosuchDebug("instantiateSprite, source != cursor");
		return;
	}

	int tm = Palette::now;
	int dt = tm - c->last_instantiate();

	if (NosuchDebugSprite) {
		NosuchDebug("instantiateSprite: tm=%d dt=%d last_instantiate=%d", tm, dt, c->last_instantiate());
	}
	if (throttle && (dt < SPRITE_THROTTLE_MS_PER_CURSOR)) {
		if (NosuchDebugSprite) {
			NosuchDebug("THROTTLE is avoiding making a new sprite at tm=%d", tm);
		}
		return;
	}

	Sprite* s = makeSprite(params.shape);
	if (s) {
		s->params.initValues(this);
		float anginit = s->params.rotanginit;
		if (s->params.rotauto) {
			anginit = -c->curr_degrees;
		}
		glm::vec2 pos;
		std::string placement = params.placement;
		if (placement == "cursor" || placement == "") {
			pos = c->curr_raw_pos;
		}
		else if( params.placement == "random" )
		{
			pos = glm::vec2( RANDFLOAT, RANDFLOAT );
		}
		else if (params.placement == "linear") {
			pos.y = 0.5;
		}
		else {
			NosuchDebug("Unexpected value for placement: %s", params.placement.c_str());
			return;
		}
		// NosuchDebug("Calling initState with movedir=%f", spriteMoveDir(c));
		s->initState(c->cid(), c->cidsource(), pos, spriteMoveDir(c), c->curr_raw_depth, anginit);
		c->set_last_instantiate(tm);
		if ( NosuchDebugSprite ) {
			NosuchDebug("Region.instantiateSprite: cid=%s", c->cid().c_str());
		}
		_spritelist->add(s,params.nsprites);
	}
}

void
Region::instantiateSpriteAt(std::string cid, glm::vec2 pos, float z) {

	// std::string shape = params.shape;
	Sprite* s = makeSprite(params.shape);
	std::string source = "instantiate_at";
	if (s) {
		s->params.initValues(this);
		float anginit = s->params.rotanginit;
		s->initState(cid, source, pos, spriteMoveDir(NULL), z, anginit);
		if (NosuchDebugSprite) {
			NosuchDebug("Region.instantiateSpriteAt: cid=%s pos=%f,%f", cid.c_str(), pos.x, pos.y);
		}
		_spritelist->add(s, params.nsprites);
	}
}

bool Region::cursorlist_lock_read() {
	int e = pthread_rwlock_rdlock(&_cursorlist_rwlock);
	if (e != 0) {
		NosuchDebug("_cursorlist_rwlock for read failed!? e=%d", e);
		return false;
	}
	NosuchDebug(2, "_cursorlist_rwlock for read succeeded");
	return true;
}

bool Region::cursorlist_lock_write() {
	int e = pthread_rwlock_wrlock(&_cursorlist_rwlock);
	if (e != 0) {
		NosuchDebug("_cursorlist_rwlock for write failed!? e=%d", e);
		return false;
	}
	NosuchDebug(2, "_cursorlist_rwlock for write succeeded");
	return true;
}

void Region::cursorlist_unlock() {
	int e = pthread_rwlock_unlock(&_cursorlist_rwlock);
	if (e != 0) {
		NosuchDebug("_cursorlist_rwlock unlock failed!? e=%d", e);
		return;
	}
	NosuchDebug(2, "_cursorlist_rwlock unlock succeeded");
}

void Region::draw(PaletteDrawer* b) {
	_spritelist->lock_read();
	_spritelist->draw(b);
	_spritelist->unlock();
}

void Region::clear() {
	_spritelist->clear();
	clearCursors();
}

void Region::advanceTo(int tm) {

	_spritelist->advanceTo(tm, params.gravity);

	if (last_tm > 0) {
		int dt = leftover_tm + tm - last_tm;
		if (dt > fire_period) {
			// NosuchDebug("Region %d calling behave->periodicFire now=%d",this->id,Palette::now);
			advanceCursorsTo(tm);
			dt -= fire_period;
		}
		leftover_tm = dt % fire_period;
	}
	last_tm = tm;
}

void Region::advanceCursorsTo(int tm) {

	if (!cursorlist_lock_read()) {
		NosuchDebug("Graphic->advanceTo returns, unable to lock cursorlist");
		return;
	}

	try {
		/*
		if (NosuchDebugCursor) {
			int sz = (int)cursors().size();
			if (sz > 0) {
				NosuchDebug("Region.advanceCursorsTo: tm=%d cursors.size=%d  sprites.size=%d", tm, cursors().size(), _spritelist->size());
			}
		}
		*/
		for (std::list<TrackedCursor*>::iterator i = cursors().begin(); i != cursors().end(); i++) {
			TrackedCursor* c = *i;

			NosuchAssert(c);
			c->advanceTo(tm);

			std::string behave = c->curr_behaviour;
			if (behave == "" || behave == "instantiate") {
				instantiateSprite(c, true);
			}
			else if (behave == "move") {
				NosuchDebug(2, "periodFire cursor move NOT IMPLEMENTED!");
			}
		}
	}
	catch (std::exception&) {
		NosuchDebug("NosuchException in advanceCursorsTo");
	}
	catch (...) {
		NosuchDebug("UNKNOWN Exception in advanceCursorsTo");
	}

	cursorlist_unlock();
}

void Region::deleteOldCursors(Palette* palette) {

	// Ideally, should only lock it for reading, and then then only lock it
	// for writing when we find an old cursor
	if (!cursorlist_lock_write()) {
		NosuchDebug("deleteOldCursors unable to lock cursorlist");
		return;
	}

	TrackedCursor* deleteCursor = NULL;
	try {
		for (std::list<TrackedCursor*>::iterator i = _cursors.begin(); i != _cursors.end(); ) {
			TrackedCursor* c = *i;
			NosuchAssert(c);
			// If the cursor hasn't been touched in a couple seconds, delete it
			int too_idle = 10 * 1000;
			if ( palette->now > (c->touched() + too_idle) ) {
				if (NosuchDebugCursor) {
					NosuchDebug("Region.deleteOldCursors: deleting cid=%s\n", c->cid().c_str());
				}
				i = _cursors.erase(i);
				delete c;
				break;
			}
			i++;
		}
	}
	catch (std::exception&) {
		NosuchDebug("NosuchException in deleteOldCursors");
	}
	catch (...) {
		NosuchDebug("UNKNOWN Exception in deleteOldCursors");
	}

	cursorlist_unlock();
}


// Scheduler* Region::scheduler() { return palette->paletteHost()->scheduler(); }
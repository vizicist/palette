#include  <stdlib.h>
#include <vector>
#include <string>
#include <cstdlib> // for srand, rand

#include "PaletteAll.h"

#define DEFINE_TYPES(t) std::vector<std::string> PlayerParams_##t##Types;
#include "PlayerParams_types.h"

Player::Player() {

	_spritelist = new SpriteList();
	_spritelistbg = new SpriteList();

	NosuchLockInit(&_player_mutex,"player");
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

void copyParamValues(PlayerParams *from, PlayerParams *to) {
#undef INIT_PARAM
#define INIT_PARAM(name,def) to->name = from->name;
#include "PlayerParams_init.h"
}

Player::~Player() {
	NosuchDebug(1,"Player DESTRUCTOR!");
}

void Player::initParams() {
	numalive = 0;
	debugcount = 0;
	last_tm = 0;
	leftover_tm = 0;
	// fire_period = 10;  // milliseconds
	fire_period = 1;  // milliseconds
	onoff = 0;
}

TrackedCursor* Player::_getTrackedCursor(std::string cid, std::string cidsource) {
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

float Player::_maxCursorDepth() {
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
Player::setTrackedCursor(Palette* palette, std::string cid, std::string cidsource, glm::vec2 pos, float z) {

	if ( pos.x < x_min || pos.x > x_max || pos.y < y_min || pos.y > y_max ) {
		NosuchDebug("Ignoring out-of-bounds cursor pos=%f,%f,%f\n",pos.x,pos.y);
		return;
	}

	if ( ! cursorlist_lock_write() ) {
		NosuchDebug("Player::setTrackedCursor, unable to lock cursorlist");
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
			NosuchDebug("Player.setTrackedCursor: new TrackedCursor cid=%s", cid.c_str());
		}
		_cursors.push_back(c);
	}
	c->touch();

	cursorlist_unlock();

	return;
}

float Player::getMoveDir(std::string movedirtype) {
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

void Player::doCursorUp(Palette* palette, std::string cid) {

	if (!cursorlist_lock_write()) {
		NosuchDebug("Player::doCursorUp, unable to lock cursorlist");
		return;
	}
	bool found = false;
	if (NosuchDebugCursor) {
		NosuchDebug("Player.doCursorUp cid=%s\n", cid.c_str());
	}
	for (std::list<TrackedCursor*>::iterator i = _cursors.begin(); i != _cursors.end(); ) {
		TrackedCursor* c = *i;
		NosuchAssert(c);
		if (NosuchDebugCursor) {
			NosuchDebug("Player.doCursorUp TrackedCursor loop c->cid=%s\n", c->cid().c_str());
		}
		if (c->cid() == cid) {
			found = true;
			if (NosuchDebugCursor) {
				NosuchDebug("Player.doCursorUp: deleting cid=%s",cid.c_str());
			}
			i = _cursors.erase(i);
			delete c;
			break;
		}
		i++;
	}
	if (NosuchDebugCursor) {
		if (!found) {
			NosuchDebug("Player.doCursorUp: didn't find cursor cid=%s", cid.c_str());
		}
		NosuchDebug("End of doCursorUp, _cursors.size = %d", _cursors.size());
	}
	cursorlist_unlock();
}

void Player::clearCursors() {

	if (!cursorlist_lock_write()) {
		NosuchDebug("Player::clearCursors, unable to lock cursorlist");
		return;
	}
	// NosuchDebug("Begin clearCursors, _cursors.size = %d",_cursors.size());
	for (std::list<TrackedCursor*>::iterator i = _cursors.begin(); i != _cursors.end(); ) {
		TrackedCursor* c = *i;
		NosuchAssert(c);
		if (NosuchDebugCursor) {
			NosuchDebug("Player.clearCursor: deleting cid=%s",c->cid().c_str());
		}
		i = _cursors.erase(i);
		delete c;
	}
	// NosuchDebug("End clearCursors, _cursors.size = %d",_cursors.size());
	cursorlist_unlock();
}

float
Player::spriteMoveDir(TrackedCursor* c)
{
	float dir;
	if (params.movedir == "cursor") {
		if (c != NULL) {
			dir = c->curr_degrees;
		}
		else {
			dir = 360.0f * RANDFLOAT;
		}
		// NosuchDebug("Player::spriteMoveDir cursor! dir=%f", dir);
		// NosuchDebug("spriteMoveDir cursor degrees = %f",c->curr_degrees);
		// not sure why I have to reverse it - the cursor values are probably reversed
		dir -= 90.0;
		if (dir < 0.0) {
			dir += 360.0;
		}
	}
	else {
		dir = getMoveDir(params.movedir);
		// NosuchDebug("Player::spriteMoveDir movedir=%s dir=%f", params.movedir.c_str(), dir);
	}
	// NosuchDebug("spriteMoveDir dir=%f movedir=%s", dir, params.movedir.c_str());
	return dir;
}

Sprite*
Player::makeSprite(std::string shape) {

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
Player::instantiateSprite(TrackedCursor* c, bool throttle) {

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
		copyParamValues(&params,&(s->params));
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
			NosuchDebug("Player.instantiateSprite: cid=%s", c->cid().c_str());
		}
		_spritelist->add(s,params.nsprites);
	}
}

void
Player::instantiateSpriteAt(std::string cid, glm::vec2 pos, float z) {

	// NosuchDebug( "Player::instantiateSpriteAt: Player=this=%lld  spritestyle=%s\n", (long long)this, this->params.spritestyle.c_str() );

	// std::string shape = params.shape;
	Sprite* s = makeSprite(params.shape);
	std::string source = "instantiate_at";
	if (s) {
		copyParamValues(&params,&(s->params));
		float anginit = s->params.rotanginit;
		s->initState(cid, source, pos, spriteMoveDir(NULL), z, anginit);
		if (NosuchDebugSprite) {
			NosuchDebug("Player.instantiateSpriteAt: cid=%s pos=%f,%f", cid.c_str(), pos.x, pos.y);
		}
		_spritelist->add(s, params.nsprites);
	}
}

void
Player::instantiateSpriteBg() {

	if( _spritelistbg->size() != 0 ) {
		return;
	}
	NosuchDebug( "Player::instantiateSpriteBg: creating square sprite for bg\n" );
	Sprite* s = makeSprite("square");
	std::string source = "bg_source";
	std::string cid = "bg_cid";
	if (s) {
		copyParamValues(&params,&(s->params));
		float anginit = 315.0f;
		glm::vec2 pos( 0.5f, 0.5f );
		float z = 0.0f;
		s->initState(cid, source, pos, spriteMoveDir(NULL), 0.5, anginit);
		if (NosuchDebugSprite) {
			NosuchDebug("Player.instantiateSpriteBg: cid=%s pos=%f,%f", cid.c_str(), pos.x, pos.y);
		}
		s->params.spritestyle = "texture";
		s->params.aspect      = 0.527f;
		s->state.size         = 3.82f;
		_spritelistbg->add(s, 100);  // should only be 1, but might someday be more
	}
}

bool Player::cursorlist_lock_read() {
	int e = pthread_rwlock_rdlock(&_cursorlist_rwlock);
	if (e != 0) {
		NosuchDebug("_cursorlist_rwlock for read failed!? e=%d", e);
		return false;
	}
	NosuchDebug(2, "_cursorlist_rwlock for read succeeded");
	return true;
}

bool Player::cursorlist_lock_write() {
	int e = pthread_rwlock_wrlock(&_cursorlist_rwlock);
	if (e != 0) {
		NosuchDebug("_cursorlist_rwlock for write failed!? e=%d", e);
		return false;
	}
	NosuchDebug(2, "_cursorlist_rwlock for write succeeded");
	return true;
}

void Player::cursorlist_unlock() {
	int e = pthread_rwlock_unlock(&_cursorlist_rwlock);
	if (e != 0) {
		NosuchDebug("_cursorlist_rwlock unlock failed!? e=%d", e);
		return;
	}
	NosuchDebug(2, "_cursorlist_rwlock unlock succeeded");
}

void Player::draw(PaletteDrawer* b) {
	_spritelist->lock_read();
	_spritelist->draw(b);
	_spritelist->unlock();
}

void Player::drawbg(PaletteDrawer* drawer) {
	if( params.inputbackground ) {
		Player::instantiateSpriteBg();
		// Only 1 sprite in _spritelistbg, so far
		// _spritelistbg->lock_read();
		_spritelistbg->draw(drawer);
		// _spritelistbg->unlock();
	}
}

void Player::clear() {
	_spritelist->clear();
	// NosuchDebug( "Player::clear is NOT (yet) clearing _spritelistbg\n" );
	clearCursors();
}

void Player::advanceTo(int tm) {

	_spritelist->advanceTo(tm, params.gravity);

	if (last_tm > 0) {
		int dt = leftover_tm + tm - last_tm;
		if (dt > fire_period) {
			// NosuchDebug("Player %d calling behave->periodicFire now=%d",this->id,Palette::now);
			advanceCursorsTo(tm);
			dt -= fire_period;
		}
		leftover_tm = dt % fire_period;
	}
	last_tm = tm;
}

void Player::advanceCursorsTo(int tm) {

	if (!cursorlist_lock_read()) {
		NosuchDebug("Graphic->advanceTo returns, unable to lock cursorlist");
		return;
	}

	try {
		/*
		if (NosuchDebugCursor) {
			int sz = (int)cursors().size();
			if (sz > 0) {
				NosuchDebug("Player.advanceCursorsTo: tm=%d cursors.size=%d  sprites.size=%d", tm, cursors().size(), _spritelist->size());
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

void Player::deleteOldCursors(Palette* palette) {

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
					NosuchDebug("Player.deleteOldCursors: deleting cid=%s\n", c->cid().c_str());
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


// Scheduler* Player::scheduler() { return palette->paletteHost()->scheduler(); }
#ifndef _PLAYER_H
#define _PLAYER_H

#include <list>

class Scheduler;
class SpriteList;
class Sprite;
class Palette;
class PaletteHost;
class PaletteDrawer;
class TrackedCursor;
class GraphicBehaviour;

#define DECLARE_TYPES(t) extern std::vector<std::string> PlayerParams_##t##Types;
#include "PlayerParams_typesdeclare.h"
void PlayerParams_InitializeTypes();

class Player;

class PlayerParams : public Params {
public:
	PlayerParams() {
#undef INIT_PARAM
#define INIT_PARAM( name, def ) ; name = ##def ;
#include "PlayerParams_init.h"
	}

	static void Initialize() {
		PlayerParams_InitializeTypes();
	}

	std::string JsonString(std::string pre, std::string indent, std::string post) {
		char* names[] = {
#include "PlayerParams_list.h"
			NULL
		};
		return JsonList(names,pre,indent,post);
	}

	void Set(std::string nm, std::string val) {
		bool stringval = false;

#define SET_DBL_PARAM(name) else if ( nm == #name ) name = string2float(val)
#define SET_FLT_PARAM(name) else if ( nm == #name ) name = float(string2float(val))
#define SET_INT_PARAM(name) else if ( nm == #name ) name = string2int(val)
#define SET_BOOL_PARAM(name) else if ( nm == #name ) name = string2bool(val)
#define SET_STR_PARAM(name) else if ( nm == #name ) (name = val),(stringval=true)

		if ( false ) { }
#include "PlayerParams_set.h"
		else {
			if (nm != "source" && nm != "player" && nm != "nuid") {
				NosuchDebug("PlayerParams::Set unrecognized param name - %s",nm.c_str());
			}
		}

		// To abide by the limits for each value, we rely on the code in Increment()
		if ( ! stringval ) {
			Increment(nm,0.0f);
		}
	}
	void Increment(std::string nm, float amount) {

#define INC_DBL_PARAM(name,mn,mx) else if (nm==#name)name=adjust(name,amount,mn,mx)
#define INC_FLT_PARAM(name,mn,mx) else if (nm==#name)name=adjust(name,amount,mn,mx)
#define INC_INT_PARAM(name,mn,mx) else if (nm==#name)name=adjust(name,amount,mn,mx)
#define INC_STR_PARAM(name,vals) else if (nm==#name)name=Params::adjust(name,amount,PlayerParams_ ## vals ## Types)
#define INC_BOOL_PARAM(name) else if (nm==#name)name=adjust(name,amount)
#define INC_NO_PARAM(name) else if (nm==#name)name=name

		if (false) {}
#include "PlayerParams_increment.h"
	}
	void Toggle(std::string nm) {
		// Just the boolean values
#define TOGGLE_PARAM(name) else if ( nm == #name ) name = ! name
		if ( false ) { }
#include "PlayerParams_toggle.h"
		else { NosuchDebug("No Toggle implemented for %s",nm.c_str()); }
	}
	std::string Get(std::string nm) {

#define GET_DBL_PARAM(name) else if(nm==#name)return DoubleString(name)
#define GET_FLT_PARAM(name) else if(nm==#name)return FloatString(name)
#define GET_INT_PARAM(name) else if(nm==#name)return IntString(name)
#define GET_BOOL_PARAM(name) else if(nm==#name)return BoolString(name)
#define GET_STR_PARAM(name) else if(nm==#name)return name

		if ( false ) { }
#include "PlayerParams_get.h"
		return "";
	}

#include "PlayerParams_declare.h"

	bool IsSpriteParam(std::string nm) {

#define IS_SPRITE_PARAM(name) if( nm == #name ) { return true; }

#include "PlayerParams_issprite.h"
		return false;
	}


};

void copyParamValues( PlayerParams* from, PlayerParams* to );

class Player {

public:
	Player();
	~Player();

	PlayerParams params;

	void initParams();
	void setTrackedCursor(Palette* palette, std::string cid, std::string cidsource, glm::vec2 pos, float z);
	float getMoveDir(std::string movedir);
	Sprite* makeSprite(std::string shape);
	void instantiateSprite(TrackedCursor* c, bool throttle);
	void instantiateSpriteAt(std::string cid, glm::vec2 pos, float z);
	void instantiateSpriteBg();
	float spriteMoveDir(TrackedCursor* c);
	// these need to be thread-safe
	void draw(PaletteDrawer* b);
	void drawbg(PaletteDrawer* b);
	void advanceTo(int tm);
	void clear();
	void deleteOldCursors(Palette* palette);

	bool cursorlist_lock_read();
	bool cursorlist_lock_write();
	void cursorlist_unlock();

	// Scheduler* scheduler();

	float _maxCursorDepth();
	size_t NumCursors() { return _cursors.size(); }

	void advanceCursorsTo(int tm);
	// void cursorDown(TrackedCursor* c);
	// void cursorDrag(TrackedCursor* c);
	// void cursorUp(TrackedCursor* c);
	void doCursorUp(Palette* palette, std::string cid);
	void clearCursors();

private:

	std::list<TrackedCursor*>& cursors() { return _cursors; }
	TrackedCursor* _getTrackedCursor(std::string cid, std::string cidsource);

	std::list<TrackedCursor*> _cursors;

	pthread_mutex_t _player_mutex;
	pthread_rwlock_t _cursorlist_rwlock;

	// Access to these lists need to be thread-safe
	// std::list<Sprite*> sprites;
	SpriteList* _spritelist;
	SpriteList* _spritelistbg; // Usually just a single sprite which is the background

	// int m_id;
	int r;
	int g;
	int b;
	int numalive;
	int onoff;
	int debugcount;

	int last_tm;
	int leftover_tm;
	int fire_period;
	// This can be adjusted to ignore things close to the edges of each area, to ignore spurious events
	float x_min;
	float y_min;
	float x_max;
	float y_max;
};

#endif

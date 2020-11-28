#ifndef _CURSOR_H
#define _CURSOR_H

// Don't instantiate a cursor's sprites more often than this number of milliseconds
#define SPRITE_THROTTLE_MS_PER_CURSOR 20

class Palette;
class Region;

class TrackedCursor {

public:

	static bool Debug;
	static bool initialized;
	static void initialize();

	TrackedCursor(Palette* palette_, std::string cid, std::string cidsource, Region* region_, glm::vec2 pos_, float z);
	~TrackedCursor();
	float radian2degree(float r) {
		return r * 360.0f / (2.0f * (float)M_PI);
	}
	bool rotauto() { return _region->params.rotauto; }

	Region* region() { return _region; }
	Palette* palette() { return _palette; }
	int touched() { return _touched; }
	void touch() { _touched = Scheduler::CurrentMilli; }
	std::string cidsource() { return _cidsource; }
	std::string cid() { return _cid; }

	float target_depth() { return _target_depth; }
	void set_target_depth(float d) { _target_depth = d; }
	// void setarea(float v) { _area = v; }

	void settargetpos(glm::vec2 p) {
		// _prev_pos = _pos;
		_target_pos = p;
	}
	// glm::vec2 previous_pos() { return _prev_pos; }

	// Manipulation of cursor-related things for graphics
	void advanceTo(int tm);

	float target_degrees() { return _target_degrees; }

	// Manipulation of cursor-related things for music

	float last_raw_depth() { return _last_raw_depth; }
	void set_last_raw_depth(float f) { _last_raw_depth = f; }

	bool isRightSide() { return ( curr_raw_pos.x >= 0.5 ); }

	std::string DebugString() {
		return NosuchSnprintf("Cursor cid=%s/%s raw=%.3f,%.3f last_raw=%.3f,%.3f target=%.3f,%.3f raw_depth=%.3f target_depth=%.3f",
			cid().c_str(),cidsource().c_str(), curr_raw_pos.x,curr_raw_pos.y, _last_raw_pos.x,_last_raw_pos.y,
			_target_pos.x,_target_pos.y,curr_raw_depth,_target_depth);
	}
	std::string DebugBrief() {
		return NosuchSnprintf("Cursor cid=%s/%s pos=%.3f,%.3f depth=%.3f",
			cid().c_str(),cidsource().c_str(), curr_raw_pos.x,curr_raw_pos.y, curr_raw_depth);
	}

	glm::vec2 curr_raw_pos;
	float curr_raw_depth;
	std::string curr_behaviour;

	float curr_degrees;

	void set_last_instantiate(int tm) { _last_instantiate = tm; }
	int last_instantiate() { return _last_instantiate; }

	int last_tm() { return _last_tm; }

private:
	// General stuff
	int _start_time;	// milliseconds
	int _last_tm;	   // milliseconds
	long _touched;   // milliseconds
	long _last_instantiate;
	Region* _region;
	Palette* _palette;
	long _lastalive;
	std::string _cid; // This is a long string, globally unique
	std::string _cidsource; // The hostname, or "sharedmem"
	// float _area;
	glm::vec2 _last_raw_pos;
	glm::vec2 _target_pos;
	// glm::vec2 _prev_pos;
	float _target_depth;
	float _last_raw_depth;
	float _smooth_degrees_factor;

	// Graphical stuff
	float _target_degrees;

	bool _g_firstdir;
};

#endif
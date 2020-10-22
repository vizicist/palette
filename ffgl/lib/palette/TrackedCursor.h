#ifndef _CURSOR_H
#define _CURSOR_H

// Don't instantiate a cursor's sprites more often than this number of milliseconds
#define SPRITE_THROTTLE_MS_PER_CURSOR 5

class Palette;
class Region;

class TrackedCursor {

public:

	static bool Debug;
	static bool initialized;
	static void initialize();

	TrackedCursor(Palette* palette_, int sidnum, std::string sidsource, Region* region_, NosuchVector pos_, double z);
	~TrackedCursor();
	double radian2degree(double r) {
		return r * 360.0 / (2.0 * (double)M_PI);
	}
	bool rotauto() { return _region->params.rotauto; }

	Region* region() { return _region; }
	Palette* palette() { return _palette; }
	int touched() { return _touched; }
	void touch() { _touched = Scheduler::CurrentMilli; }
	std::string sidsource() { return _sidsource; }
	int sidnum() { return _sidnum; }
	// double area() { return _area; }

	double target_depth() { return _target_depth; }
	void set_target_depth(double d) { _target_depth = d; }
	// void setarea(double v) { _area = v; }

	void settargetpos(NosuchVector p) {
		// _prev_pos = _pos;
		_target_pos = p;
	}
	// NosuchVector previous_pos() { return _prev_pos; }

	// Manipulation of cursor-related things for graphics
	void advanceTo(int tm);

	double target_degrees() { return _target_degrees; }

	// Manipulation of cursor-related things for music

	double last_raw_depth() { return _last_raw_depth; }
	void set_last_raw_depth(double f) { _last_raw_depth = f; }

	bool isRightSide() { return ( curr_raw_pos.x >= 0.5 ); }

	std::string DebugString() {
		return NosuchSnprintf("Cursor sid=%d/%s raw=%.3f,%.3f last_raw=%.3f,%.3f target=%.3f,%.3f raw_depth=%.3f target_depth=%.3f",
			sidnum(),sidsource().c_str(), curr_raw_pos.x,curr_raw_pos.y, _last_raw_pos.x,_last_raw_pos.y,
			_target_pos.x,_target_pos.y,curr_raw_depth,_target_depth);
	}
	std::string DebugBrief() {
		return NosuchSnprintf("Cursor sid=%d/%s pos=%.3f,%.3f depth=%.3f",
			sidnum(),sidsource().c_str(), curr_raw_pos.x,curr_raw_pos.y, curr_raw_depth);
	}

	NosuchVector curr_raw_pos;
	double curr_raw_depth;
	std::string curr_behaviour;

	double curr_degrees;

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
	int _sidnum; // This is the raw sid, e.g. 4000
	std::string _sidsource; // The hostname, or "sharedmem"
	// double _area;
	NosuchVector _last_raw_pos;
	NosuchVector _target_pos;
	// NosuchVector _prev_pos;
	double _target_depth;
	double _last_raw_depth;
	double _smooth_degrees_factor;

	// Graphical stuff
	double _target_degrees;

	bool _g_firstdir;
};

#endif
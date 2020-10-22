#include "PaletteAll.h"

bool TrackedCursor::initialized = false;
bool TrackedCursor::Debug = false;

// std::vector<std::string> TrackedCursor::behaviourTypes;

TrackedCursor::TrackedCursor(Palette* palette_, int sidnum, std::string sidsource, Region* region_, NosuchVector pos_, double z) {
	// _area = area_;
	_palette = palette_;
	// _last_pitches.clear();
	// _last_channel = -1;
	// _last_click = -1;
	_sidsource = sidsource;
	_sidnum = sidnum;
	_region = region_;

	if ( Debug) {
		NosuchDebug("NEW TrackedCursor region=%d sidnum=%d source=%s",_region,_sidnum,_sidsource.c_str());
	}

	curr_raw_pos = pos_;
	// NosuchDebug("Setting curr_raw_pos and _last_raw_pos to %f %f",curr_raw_pos.x,curr_raw_pos.y);

	_target_pos = pos_;
	_last_raw_pos = pos_;

	// _prev_pos = pos_;
	_last_raw_depth = z;
	curr_raw_depth = z;
	_target_depth = z;
	NosuchDebug(2,"Cursor CONSTRUCTOR sid=%d/%s  pos_ = %.3f %.3f",_sidnum,_sidsource.c_str(),pos_.x,pos_.y);
	_start_time = Palette::now;
	_last_tm = _start_time;
	_last_instantiate = 0;

	curr_degrees = 0.0f;
	_smooth_degrees_factor = 0.2f;
	// _target_degrees = 0.0f;

	_g_firstdir = true;
	_touched = 0;
}

void TrackedCursor::initialize() {

	if ( initialized )
		return;
	initialized = true;

	// behaviourTypes.push_back("instantiate"); // instantiate sprites continuously
	// behaviourTypes.push_back("move");        // instantiate and then move a single sprite
	// behaviourTypes.push_back("accumulate");  // continuously accumulate points for a single sprite
}

TrackedCursor::~TrackedCursor() {
	if ( Debug ) {
		NosuchDebug("TrackedCursor DESTRUCTOR!!  this=%llx _sid=%d/%s",(long long)this,_sidnum,_sidsource.c_str());
	}
}

double
normalize_degrees(double d) {
	if ( d < 0.0f ) {
		d += 360.0f;
	} else if ( d > 360.0f ) {
		d -= 360.0f;
	}
	return d;
}

void
TrackedCursor::advanceTo(int tm) {

	int dt = tm - _last_tm;
	if ( dt <= 0 ) {
		return;
	}

	if ( curr_raw_pos.x == _target_pos.x && curr_raw_pos.y == _target_pos.y ) {
		NosuchDebug(1,"Cursor::advanceTo, current and target are the same");
		return;
	}

	NosuchVector dpos = _target_pos.sub(curr_raw_pos);
	double raw_distance = dpos.mag();

	// Not sure why I have this here, maybe just for sanity check
	if ( raw_distance > 1.5f ) {
		NosuchDebug("Cursor::advanceTo, raw_distance>1.5 !?");
		return;
	}
	if ( raw_distance == 0.0f ) {
		NosuchDebug("Cursor::advanceTo, raw_distance=0.0 !?");
		return;
	}

	_last_raw_pos = curr_raw_pos;

	int smoothxyz_factor = 1 + region()->params.smoothxyz;

	dpos = dpos.normalize();

	// curr_raw_pos = curr_raw_pos.sub(dpos);
	// curr_raw_pos = _target_pos;

	double old_x = curr_raw_pos.x;
	curr_raw_pos.x = ((smoothxyz_factor-1)*curr_raw_pos.x + _target_pos.x) / (double)smoothxyz_factor;
	curr_raw_pos.y = ((smoothxyz_factor-1)*curr_raw_pos.y + _target_pos.y) / (double)smoothxyz_factor;

	// XXX - should be taking time (dt) into account
	curr_raw_depth = ((smoothxyz_factor-1)*curr_raw_depth + target_depth()) / (double)smoothxyz_factor;

	/////////////// smooth the degrees
	double tooshort = 0.01f; // 0.05f;
	if (raw_distance < tooshort) {
		// NosuchDebug("   raw_distance=%.3f too small %s\n",
		// 	raw_distance,DebugString().c_str());
	} else {
		NosuchVector dp = curr_raw_pos.sub(_last_raw_pos);
		double heading = dp.heading();
		// NosuchDebug("");
		// NosuchDebug("HEADING %f",heading);
		_target_degrees = radian2degree(heading);
		_target_degrees += 90.0;
		_target_degrees = normalize_degrees(_target_degrees);

		if (_g_firstdir) {
			curr_degrees = _target_degrees;
			_g_firstdir = false;
		} else {
			double dd1 = _target_degrees - curr_degrees;
			double dd;
			if ( dd1 > 0.0f ) {
				if ( dd1 > 180.0f ) {
					dd = -(360.0f - dd1);
				}
				else {
					dd = dd1;
				}
			} else {
				if ( dd1 < -180.0f ) {
					dd = dd1 + 360.0f;
				}
				else {
					dd = dd1;
				}
			}
			// double smooth_degrees_factor = _smooth_degrees_factor;
			double smooth_degrees_factor = 1.0f / (smoothxyz_factor * 10);

			curr_degrees = curr_degrees + (dd*smooth_degrees_factor);
			curr_degrees = normalize_degrees(curr_degrees);
		}
		// NosuchDebug("   FINAL current_degrees =%.4f",curr_degrees);
	}

	_last_tm = Palette::now;

	// NosuchDebug("   end of advanceTo %s",DebugString().c_str());
}
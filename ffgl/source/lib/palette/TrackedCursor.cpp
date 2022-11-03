#include "PaletteAll.h"

bool TrackedCursor::initialized = false;
bool TrackedCursor::Debug = false;

// std::vector<std::string> TrackedCursor::behaviourTypes;

TrackedCursor::TrackedCursor(Palette* palette_, std::string cid, std::string cidsource, Player* player_, glm::vec2 pos_, float z) {
	// _area = area_;
	_palette = palette_;
	// _last_pitches.clear();
	// _last_channel = -1;
	// _last_click = -1;
	_cidsource = cidsource;
	_cid = cid;
	_player = player_;

	if ( Debug) {
		NosuchDebug("NEW TrackedCursor player=%d cid=%s source=%s",_player, cid.c_str(),_cidsource.c_str());
	}

	curr_raw_pos = pos_;
	// NosuchDebug("Setting curr_raw_pos and _last_raw_pos to %f %f",curr_raw_pos.x,curr_raw_pos.y);

	_target_pos = pos_;
	_last_raw_pos = pos_;

	// _prev_pos = pos_;
	_last_raw_depth = z;
	curr_raw_depth = z;
	_target_depth = z;
	NosuchDebug(2,"Cursor CONSTRUCTOR cid=%s/%s  pos_ = %.3f %.3f",_cid.c_str(),_cidsource.c_str(),pos_.x,pos_.y);
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
		NosuchDebug("TrackedCursor DESTRUCTOR!!  this=%llx _cid=%s/%s",(long long)this,_cid.c_str(),_cidsource.c_str());
	}
}

float
normalize_degrees(float d) {
	if ( d < 0.0f ) {
		d += 360.0f;
	} else if ( d > 360.0f ) {
		d -= 360.0f;
	}
	return d;
}

static float headingOf(glm::vec2 point) {
	return -atan2(-point[1], point[0]);
}

void TrackedCursor::advanceTo( int tm )
{
	int dt = tm - _last_tm;
	if( dt <= 0 )
	{
		return;
	}

	if( curr_raw_pos.x == _target_pos.x && curr_raw_pos.y == _target_pos.y )
	{
		NosuchDebug( 1, "Cursor::advanceTo, current and target are the same" );
		return;
	}

	glm::vec2 dpos      = _target_pos - curr_raw_pos;
	float raw_distance = glm::length( dpos );

	// Not sure why I have this here, maybe just for sanity check
	if( raw_distance > 1.5f )
	{
		NosuchDebug( "Cursor::advanceTo, raw_distance>1.5 !?" );
		return;
	}
	if( raw_distance == 0.0f )
	{
		NosuchDebug( "Cursor::advanceTo, raw_distance=0.0 !?" );
		return;
	}

	_last_raw_pos = curr_raw_pos;

	int smoothxyz_factor = 1 + player()->params.smoothxyz;

	dpos = glm::normalize( dpos );

	// curr_raw_pos = curr_raw_pos.sub(dpos);
	// curr_raw_pos = _target_pos;

	curr_raw_pos = glm::vec2(
		( ( smoothxyz_factor - 1 ) * curr_raw_pos.x + _target_pos.x ) / (float)smoothxyz_factor,
		( ( smoothxyz_factor - 1 ) * curr_raw_pos.y + _target_pos.y ) / (float)smoothxyz_factor );

	// XXX - should be taking time (dt) into account
	curr_raw_depth = ((smoothxyz_factor-1)*curr_raw_depth + target_depth()) / (float)smoothxyz_factor;

	/////////////// smooth the degrees
	float tooshort = 0.01f; // 0.05f;
	if (raw_distance < tooshort) {
		// NosuchDebug("   raw_distance=%.3f too small %s\n",
		// 	raw_distance,DebugString().c_str());
	} else {
		glm::vec2 dp = curr_raw_pos - _last_raw_pos;
		float heading = headingOf(dp);
		// NosuchDebug("");
		// NosuchDebug("HEADING %f",heading);
		_target_degrees = radian2degree(heading);
		_target_degrees += 90.0;
		_target_degrees = normalize_degrees(_target_degrees);

		if (_g_firstdir) {
			curr_degrees = _target_degrees;
			_g_firstdir = false;
		} else {
			float dd1 = _target_degrees - curr_degrees;
			float dd;
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
			// float smooth_degrees_factor = _smooth_degrees_factor;
			float smooth_degrees_factor = 1.0f / (smoothxyz_factor * 10);

			curr_degrees = curr_degrees + (dd*smooth_degrees_factor);
			curr_degrees = normalize_degrees(curr_degrees);
		}
		// NosuchDebug("   FINAL current_degrees =%.4f",curr_degrees);
	}

	_last_tm = Palette::now;

	// NosuchDebug("   end of advanceTo %s",DebugString().c_str());
}
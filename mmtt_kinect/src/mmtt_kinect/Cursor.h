#ifndef _CURSOR_H
#define _CURSOR_H

// Don't instantiate a cursor's sprites more often than this number of milliseconds
#define SPRITE_THROTTLE_MS_PER_CURSOR 5

class Region;

class Cursor {

public:

	Cursor(std::string region, int sidnum, NosuchVector pos_, double depth_, double area_);
	~Cursor();

	std::string region;
	int sidnum; // This is the raw sid, e.g. 4000
	NosuchVector curr_pos;
	double curr_depth;
};

#endif
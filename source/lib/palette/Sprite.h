#ifndef _SPRITE_H
#define _SPRITE_H

// Note - this Params class is different from PaletteParams and RegionParams because
// it doesn't need the Set/Increment/Toggle methods.

class SpriteParams {

public:

	void initValues(Region* r) {

		RegionParams& rp = r->params;

#define INIT_PARAM(name) name = rp.##name;

#include "SpriteParams_init.h"
	}

#include "SpriteParams_declare.h"
};

class SpriteList {

public:
	SpriteList();
	void lock_read();
	void lock_write();
	void unlock();
	void draw(PaletteHost* b);
	void clear();
	void advanceTo(int tm, int gravity);
	void computeForce(std::list<Sprite*> &sprites, int gravity);
	void add(Sprite* s, int limit);
	int size() {
		return (int)(sprites.size());
	}

private:
	std::list<Sprite*> sprites;
	pthread_rwlock_t rwlock;

};

class SpriteState {
public:
	SpriteState() {
		visible = false;
		direction = 0.0;
		hue = 0.0f;
		huefill = 0.0f;
		pos = NosuchVector(0.0f,0.0f);
		depth = 0.0;
		size = 0.5;
		alpha = 1.0;
		born = 0;
		last_tm = 0;
		killme = false;
		rotangsofar = 0.0f;
		stationary = false;
		sidnum = 0;
		sidsource = "nosuchsource";
		rotanginit = 0.0;
		gravityForce = NosuchVector(0.0, 0.0);
	}
	bool visible;
	double direction;
	double hue;
	double huefill;
	NosuchVector pos;
	double depth;
	double size;
	double alpha;
	int born;
	int last_tm;
	bool killme;
	double rotangsofar;
	bool stationary;
	int sidnum;
	std::string sidsource;
	int seq;     // sprite sequence # (mostly for debugging)
	int rotdir;  // -1, 0, 1
	double rotanginit;
	NosuchVector gravityForce;
};

class Sprite {
	
public:

	Sprite();
	virtual ~Sprite();

	virtual void drawShape(PaletteHost* app, int xdir, int ydir) = 0;
	virtual bool fixedScale() { return false; }
	virtual void startAccumulate(TrackedCursor* c) { };
	virtual void accumulate(TrackedCursor* c) { }

	// virtual double width() { return 1.0; }
	// virtual double height() { return 1.0; }
	virtual double width() { return state.size * state.depth; }
	virtual double height() { return state.size * state.depth; }

	static bool initialized;
	static void initialize();
	// static std::vector<std::string> spriteShapes;
	static double degree2radian(double deg);

	void initState(int sidnum, std::string sidsource, NosuchVector& pos, double movedir, double depth, double rotanginit);

	// Screen space is 2.0x2.0, while cursor space is 1.0x1.0
	void scaleCursorSpaceToScreenSpace(NosuchVector& pos) {
		state.pos.x *= 2.0f;
		state.pos.y *= 2.0f;
	}

	void draw(PaletteHost* app);
	void drawAt(PaletteHost* app, double x,double y, double w, double h, int xdir, int ydir);
	NosuchVector deltaInDirection(double dt, double dir, double speed);
	int rotangdirOf(std::string s);
	void advanceTo(int tm, NosuchVector force);

	SpriteParams params;
	SpriteState state;

protected:
	// Sprite(int sidnum, std::string sidsource, Region* r);
	double vertexNoise();

	static int NextSeq;

private:
	void draw(PaletteHost* app, double scaled_z);
};

class SpriteSquare : public Sprite {

public:
	SpriteSquare();
	void drawShape(PaletteHost* app, int xdir, int ydir);

private:
	bool noise_initialized;
	double noise_x0;
	double noise_y0;
	double noise_x1;
	double noise_y1;
	double noise_x2;
	double noise_y2;
	double noise_x3;
	double noise_y3;
};

class SpriteTriangle : public Sprite {

public:
	SpriteTriangle();
	void drawShape(PaletteHost* app, int xdir, int ydir);

private:
	bool noise_initialized;
	double noise_x0;
	double noise_y0;
	double noise_x1;
	double noise_y1;
	double noise_x2;
	double noise_y2;
};

class SpriteCircle : public Sprite {

public:
	SpriteCircle();
	void drawShape(PaletteHost* app, int xdir, int ydir);
};

class SpriteArc : public Sprite {

public:
	SpriteArc();
	void drawShape(PaletteHost* app, int xdir, int ydir);
};

class SpriteLine : public Sprite {

public:
	SpriteLine();
	void drawShape(PaletteHost* app, int xdir, int ydir);

private:
	bool noise_initialized;
	double noise_x0;
	double noise_y0;
	double noise_x1;
	double noise_y1;
};

#endif
